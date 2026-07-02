package ws

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"syncslate/internal/message"
)

type Router struct {
	hub        *Hub
	msgService *message.Service
	db         *sql.DB
}

func NewRouter(hub *Hub, msgService *message.Service, db *sql.DB) *Router {
	return &Router{hub: hub, msgService: msgService, db: db}
}

func (r *Router) HandleEvent(client *Client, raw []byte) {
	var evt Event
	if err := json.Unmarshal(raw, &evt); err != nil {
		return
	}

	switch evt.Type {
	case EventPing:
		client.SendJSON(EventPong, map[string]string{"status": "ok"})

	case EventMessageSend:
		var p MessageSendPayload
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return
		}
		r.handleMessageSend(client, p)

	case EventMessageEdit:
		var p MessageEditPayload
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return
		}
		r.handleMessageEdit(client, p)

	case EventMessageDelete:
		var p MessageDeletePayload
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return
		}
		r.handleMessageDelete(client, p)

	case EventChatRead:
		var p ChatReadPayload
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return
		}
		r.handleChatRead(client, p)

	case EventChatTyping:
		var p ChatTypingPayload
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return
		}
		r.handleChatTyping(client, p)
	}
}

func (r *Router) handleMessageSend(client *Client, p MessageSendPayload) {
	msg, err := r.msgService.Create(p.ChatID, client.userID, p.Content, p.MediaID, p.ReplyToID)
	if err != nil {
		client.SendJSON("error", map[string]string{"message": err.Error()})
		return
	}

	// Ack sender
	client.SendJSON(EventMessageAck, map[string]interface{}{
		"client_msg_id": p.ClientMsgID,
		"server_msg_id": msg.ID,
		"timestamp":     msg.CreatedAt.UnixMilli(),
	})

	// Broadcast to chat members
	r.hub.BroadcastToChat(p.ChatID, client.userID, EventMessageNew, msg)
}

func (r *Router) handleMessageEdit(client *Client, p MessageEditPayload) {
	msg, err := r.msgService.EditMessage(p.MessageID, client.userID, p.Content)
	if err != nil {
		client.SendJSON("error", map[string]string{"message": err.Error()})
		return
	}

	r.hub.BroadcastToChat(msg.ChatID, "", EventMessageEdited, map[string]interface{}{
		"message_id":  msg.ID,
		"chat_id":     msg.ChatID,
		"new_content": msg.Content,
		"edited_at":   msg.EditedAt,
	})
}

func (r *Router) handleMessageDelete(client *Client, p MessageDeletePayload) {
	msg, err := r.msgService.DeleteMessage(p.MessageID, client.userID, p.ForEveryone)
	if err != nil {
		client.SendJSON("error", map[string]string{"message": err.Error()})
		return
	}

	if p.ForEveryone {
		// Announce delete event (Ayu Spy on client intercepts this!)
		r.hub.BroadcastToChat(msg.ChatID, "", EventMessageDeleted, map[string]interface{}{
			"message_id": msg.ID,
			"chat_id":    msg.ChatID,
			"deleted_at": msg.DeletedAt,
		})
	}
}

// Server-side Ghost Mode Enforcement for Read Receipts
func (r *Router) handleChatRead(client *Client, p ChatReadPayload) {
	var ghostMode int
	_ = r.db.QueryRow(`SELECT ghost_mode FROM users WHERE id = ?`, client.userID).Scan(&ghostMode)

	nowMs := time.Now().UnixMilli()

	// Update local read receipt in database
	_, _ = r.db.Exec(`INSERT INTO read_receipts (chat_id, user_id, last_read_message_id, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(chat_id, user_id) DO UPDATE SET last_read_message_id = excluded.last_read_message_id, updated_at = excluded.updated_at`,
		p.ChatID, client.userID, p.LastReadMessageID, nowMs)

	// GHOST MODE ENFORCEMENT: do not broadcast if user is in Ghost Mode!
	if ghostMode == 1 {
		slog.Debug("ghost mode active: suppressing read receipt broadcast", "user_id", client.userID)
		return
	}

	r.hub.BroadcastToChat(p.ChatID, client.userID, EventChatReadUpdate, map[string]interface{}{
		"chat_id":              p.ChatID,
		"user_id":              client.userID,
		"last_read_message_id": p.LastReadMessageID,
	})
}

// Server-side Ghost Mode Enforcement for Typing Indicators
func (r *Router) handleChatTyping(client *Client, p ChatTypingPayload) {
	var ghostMode int
	_ = r.db.QueryRow(`SELECT ghost_mode FROM users WHERE id = ?`, client.userID).Scan(&ghostMode)

	// GHOST MODE ENFORCEMENT: silently drop typing event!
	if ghostMode == 1 {
		slog.Debug("ghost mode active: suppressing typing indicator broadcast", "user_id", client.userID)
		return
	}

	r.hub.BroadcastToChat(p.ChatID, client.userID, EventChatTypingUpdate, map[string]interface{}{
		"chat_id":   p.ChatID,
		"user_id":   client.userID,
		"is_typing": p.IsTyping,
	})
}
