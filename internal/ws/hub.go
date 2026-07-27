package ws

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"sync"
)

type Hub struct {
	db      *sql.DB
	clients sync.Map // userID -> map[*Client]bool
	mu      sync.Mutex
}

func NewHub(db *sql.DB) *Hub {
	return &Hub{db: db}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	val, _ := h.clients.Load(c.userID)
	var clientMap map[*Client]bool
	if val == nil {
		clientMap = make(map[*Client]bool)
	} else {
		clientMap = val.(map[*Client]bool)
	}

	clientMap[c] = true
	h.clients.Store(c.userID, clientMap)

	slog.Info("ws client connected", "user_id", c.userID)

	// Send WS sync lifecycle events
	c.SendJSON("sync:start", map[string]interface{}{"status": "syncing", "timestamp": time.Now().UnixMilli()})
	c.SendJSON("sync:complete", map[string]interface{}{"status": "synced", "timestamp": time.Now().UnixMilli()})

	// Update last_seen & broadcast presence online if not Ghost Mode
	go h.handlePresenceChange(c.userID, "online")
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	val, exists := h.clients.Load(c.userID)
	if !exists {
		return
	}

	clientMap := val.(map[*Client]bool)
	delete(clientMap, c)

	if len(clientMap) == 0 {
		h.clients.Delete(c.userID)
		slog.Info("ws user disconnected completely", "user_id", c.userID)
		go h.handlePresenceChange(c.userID, "offline")
	} else {
		h.clients.Store(c.userID, clientMap)
	}
}

func (h *Hub) IsUserOnline(userID string) bool {
	val, exists := h.clients.Load(userID)
	if !exists || val == nil {
		return false
	}
	clientMap := val.(map[*Client]bool)
	return len(clientMap) > 0
}

func (h *Hub) SendToUser(userID string, eventType string, payload interface{}) {
	val, exists := h.clients.Load(userID)
	if !exists {
		return
	}

	clientMap := val.(map[*Client]bool)
	for c := range clientMap {
		c.SendJSON(eventType, payload)
	}
}

func (h *Hub) BroadcastToChat(chatID string, excludeUserID string, eventType string, payload interface{}) {
	rows, err := h.db.Query(`SELECT user_id FROM chat_members WHERE chat_id = ?`, chatID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err == nil {
			if uid != excludeUserID {
				h.SendToUser(uid, eventType, payload)
			}
		}
	}
}

func (h *Hub) handlePresenceChange(userID, status string) {
	nowMs := time.Now().UnixMilli()
	_, _ = h.db.Exec(`UPDATE users SET last_seen = ? WHERE id = ?`, nowMs, userID)

	var ghostMode int
	_ = h.db.QueryRow(`SELECT ghost_mode FROM users WHERE id = ?`, userID).Scan(&ghostMode)

	// Ghost mode server-side enforcement: suppress presence changes!
	if ghostMode == 1 {
		return
	}

	// Fetch all chat peers
	rows, err := h.db.Query(`
		SELECT DISTINCT cm2.user_id
		FROM chat_members cm1
		JOIN chat_members cm2 ON cm1.chat_id = cm2.chat_id
		WHERE cm1.user_id = ? AND cm2.user_id != ?`, userID, userID)
	if err != nil {
		return
	}
	defer rows.Close()

	payload := map[string]interface{}{
		"user_id":   userID,
		"status":    status,
		"last_seen": nowMs,
		"is_online": status == "online",
	}

	for rows.Next() {
		var peerID string
		if err := rows.Scan(&peerID); err == nil {
			h.SendToUser(peerID, EventPresenceChange, payload)
		}
	}
}

func marshalEvent(eventType string, payload interface{}) ([]byte, error) {
	pBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	evt := Event{
		Type:    eventType,
		Payload: pBytes,
	}
	return json.Marshal(evt)
}
