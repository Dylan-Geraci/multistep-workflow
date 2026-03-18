package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dylangeraci/flowforge/internal/model"
	"github.com/gorilla/websocket"
)

func TestHubClientLifecycle(t *testing.T) {
	hub := NewHub(nil, metrics.New())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Start a test WebSocket server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade error: %v", err)
			return
		}
		client := hub.NewClient(conn, "user-1")
		hub.Register(client)
		go client.WritePump()
		client.ReadPump()
	}))
	defer srv.Close()

	// Connect
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	// Give hub time to register
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	clientCount := len(hub.clients)
	hub.mu.RUnlock()
	if clientCount != 1 {
		t.Errorf("expected 1 client, got %d", clientCount)
	}

	// Subscribe to a run
	sub := model.WSIncomingMessage{Type: "subscribe", RunIDs: []string{"run-123"}}
	data, _ := json.Marshal(sub)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("write error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	subCount := len(hub.runSubs["run-123"])
	hub.mu.RUnlock()
	if subCount != 1 {
		t.Errorf("expected 1 subscriber for run-123, got %d", subCount)
	}

	// Unsubscribe
	unsub := model.WSIncomingMessage{Type: "unsubscribe", RunIDs: []string{"run-123"}}
	data, _ = json.Marshal(unsub)
	conn.WriteMessage(websocket.TextMessage, data)
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	subCount = len(hub.runSubs["run-123"])
	hub.mu.RUnlock()
	if subCount != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", subCount)
	}

	// Disconnect
	conn.Close()
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	clientCount = len(hub.clients)
	hub.mu.RUnlock()
	if clientCount != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", clientCount)
	}
}

func TestHubMessageRouting(t *testing.T) {
	hub := NewHub(nil, metrics.New())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := hub.NewClient(conn, "user-2")
		hub.Register(client)
		go client.WritePump()
		client.ReadPump()
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// Subscribe
	sub := model.WSIncomingMessage{Type: "subscribe", RunIDs: []string{"run-456"}}
	data, _ := json.Marshal(sub)
	conn.WriteMessage(websocket.TextMessage, data)
	time.Sleep(50 * time.Millisecond)

	// Simulate sending an event directly to the hub's subscription
	event := model.WSEvent{
		Type:  model.EventRunCompleted,
		RunID: "run-456",
		Data:  json.RawMessage(`{"status":"completed"}`),
	}
	eventData, _ := json.Marshal(event)

	hub.mu.RLock()
	for client := range hub.runSubs["run-456"] {
		client.send <- eventData
	}
	hub.mu.RUnlock()

	// Read the message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var received model.WSEvent
	if err := json.Unmarshal(msg, &received); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if received.Type != model.EventRunCompleted {
		t.Errorf("expected event type %s, got %s", model.EventRunCompleted, received.Type)
	}
	if received.RunID != "run-456" {
		t.Errorf("expected run_id run-456, got %s", received.RunID)
	}
}
