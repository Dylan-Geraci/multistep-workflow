package handler

import (
	"log/slog"
	"net/http"

	"github.com/dylangeraci/flowforge/internal/middleware"
	"github.com/dylangeraci/flowforge/internal/ws"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub *ws.Hub
}

func NewWSHandler(hub *ws.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

func (h *WSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "ws upgrade error", "error", err)
		return
	}

	client := h.hub.NewClient(conn, userID)
	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
