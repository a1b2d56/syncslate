package ws

import (
	"encoding/json"
	"net/http"
	"time"

	"syncslate/internal/auth"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow local connections
	},
}

type Handler struct {
	hub         *Hub
	router      *Router
	authService *auth.Service
}

func NewHandler(hub *Hub, router *Router, authService *auth.Service) *Handler {
	return &Handler{hub: hub, router: router, authService: authService}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// 5-second auth handshake timeout
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return
	}

	var evt Event
	if err := jsonUnmarshal(message, &evt); err != nil || evt.Type != EventAuth {
		conn.WriteJSON(map[string]interface{}{"type": EventAuthError, "message": "invalid handshake envelope"})
		conn.Close()
		return
	}

	var authPayload AuthPayload
	if err := jsonUnmarshal(evt.Payload, &authPayload); err != nil {
		conn.WriteJSON(map[string]interface{}{"type": EventAuthError, "message": "invalid auth payload"})
		conn.Close()
		return
	}

	userID, err := h.authService.ValidateToken(authPayload.Token)
	if err != nil {
		conn.WriteJSON(map[string]interface{}{"type": EventAuthError, "message": "invalid or expired token"})
		conn.Close()
		return
	}

	// Auth ack
	conn.WriteJSON(map[string]interface{}{
		"type":    EventAuthAck,
		"payload": map[string]string{"status": "ok", "user_id": userID},
	})

	client := NewClient(h.hub, conn, userID)
	h.hub.Register(client)

	go client.WritePump()
	client.ReadPump(h.router)
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
