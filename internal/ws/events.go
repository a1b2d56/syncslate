package ws

import "encoding/json"

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

const (
	EventAuth             = "auth"
	EventAuthAck          = "auth:ack"
	EventAuthError        = "auth:error"
	EventPing             = "ping"
	EventPong             = "pong"
	EventMessageSend      = "message:send"
	EventMessageNew       = "message:new"
	EventMessageAck       = "message:ack"
	EventMessageEdit      = "message:edit"
	EventMessageEdited    = "message:edited"
	EventMessageDelete    = "message:delete"
	EventMessageDeleted   = "message:deleted"
	EventChatRead         = "chat:read"
	EventChatReadUpdate   = "chat:read_update"
	EventChatTyping       = "chat:typing"
	EventChatTypingUpdate = "chat:typing_update"
	EventPresenceUpdate   = "presence:update"
	EventPresenceChange   = "presence:change"
	EventMessageReqNew    = "message_request:new"
)

type AuthPayload struct {
	Token string `json:"token"`
}

type MessageSendPayload struct {
	ClientMsgID string  `json:"client_msg_id"`
	ChatID      string  `json:"chat_id"`
	Content     string  `json:"content"`
	MediaID     *string `json:"media_id,omitempty"`
	ReplyToID   *string `json:"reply_to_id,omitempty"`
}

type MessageEditPayload struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
}

type MessageDeletePayload struct {
	MessageID   string `json:"message_id"`
	ForEveryone bool   `json:"for_everyone"`
}

type ChatReadPayload struct {
	ChatID            string `json:"chat_id"`
	LastReadMessageID string `json:"last_read_message_id"`
}

type ChatTypingPayload struct {
	ChatID   string `json:"chat_id"`
	IsTyping bool   `json:"is_typing"`
}

type PresenceUpdatePayload struct {
	Status string `json:"status"` // online, offline
}
