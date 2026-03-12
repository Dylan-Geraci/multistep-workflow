package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/dylangeraci/flowforge/internal/model"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	userID string
	send   chan []byte
	subs   map[string]bool // subscribed run IDs
	mu     sync.Mutex
}

type Hub struct {
	clients    map[*Client]bool
	runSubs    map[string]map[*Client]bool // runID → set of clients
	register   chan *Client
	unregister chan *Client
	subscribe  chan subscribeMsg
	rdb        *redis.Client
	mu         sync.RWMutex
}

type subscribeMsg struct {
	client *Client
	runIDs []string
	unsub  bool
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		runSubs:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		subscribe:  make(chan subscribeMsg, 64),
		rdb:        rdb,
	}
}

func (h *Hub) Run(ctx context.Context) {
	// Start Redis PSubscribe listener
	go h.listenRedis(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				// Remove from all subscriptions
				for runID := range client.subs {
					if subs, ok := h.runSubs[runID]; ok {
						delete(subs, client)
						if len(subs) == 0 {
							delete(h.runSubs, runID)
						}
					}
				}
			}
			h.mu.Unlock()
		case msg := <-h.subscribe:
			h.mu.Lock()
			for _, runID := range msg.runIDs {
				if msg.unsub {
					if subs, ok := h.runSubs[runID]; ok {
						delete(subs, msg.client)
						if len(subs) == 0 {
							delete(h.runSubs, runID)
						}
					}
					msg.client.mu.Lock()
					delete(msg.client.subs, runID)
					msg.client.mu.Unlock()
				} else {
					if _, ok := h.runSubs[runID]; !ok {
						h.runSubs[runID] = make(map[*Client]bool)
					}
					h.runSubs[runID][msg.client] = true
					msg.client.mu.Lock()
					msg.client.subs[runID] = true
					msg.client.mu.Unlock()
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) listenRedis(ctx context.Context) {
	if h.rdb == nil {
		return
	}
	pubsub := h.rdb.PSubscribe(ctx, "flowforge:events:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			// Extract runID from channel name: "flowforge:events:<runID>"
			runID := msg.Channel[len("flowforge:events:"):]

			h.mu.RLock()
			clients := h.runSubs[runID]
			for client := range clients {
				select {
				case client.send <- []byte(msg.Payload):
				default:
					// Client buffer full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) NewClient(conn *websocket.Conn, userID string) *Client {
	return &Client{
		hub:    h,
		conn:   conn,
		userID: userID,
		send:   make(chan []byte, 256),
		subs:   make(map[string]bool),
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Error("websocket read error", "error", err)
			}
			break
		}

		var msg model.WSIncomingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "subscribe":
			c.hub.subscribe <- subscribeMsg{client: c, runIDs: msg.RunIDs}
		case "unsubscribe":
			c.hub.subscribe <- subscribeMsg{client: c, runIDs: msg.RunIDs, unsub: true}
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
