package chat

import (
	"encoding/json"
	"net/http"
	"strconv"

	"syncslate/internal/middleware"
	"syncslate/internal/ws"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
	hub     *ws.Hub
}

func NewHandler(service *Service, hub *ws.Hub) *Handler {
	return &Handler{service: service, hub: hub}
}

func (h *Handler) ListChats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	chats, err := h.service.GetUserChats(userID)
	if err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to fetch chats"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}

func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	chatID := chi.URLParam(r, "id")
	userID := middleware.GetUserIDFromContext(r.Context())
	before := r.URL.Query().Get("before")
	limitStr := r.URL.Query().Get("limit")

	limit := 50
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}

	messages, err := h.service.GetChatMessages(chatID, userID, before, limit)
	if err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to fetch messages"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

type muteReq struct {
	MuteUntil *int64 `json:"muteUntil"`
}

func (h *Handler) Mute(w http.ResponseWriter, r *http.Request) {
	chatID := chi.URLParam(r, "id")
	userID := middleware.GetUserIDFromContext(r.Context())

	var req muteReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := h.service.MuteChat(chatID, userID, req.MuteUntil); err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to mute chat"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) Unmute(w http.ResponseWriter, r *http.Request) {
	chatID := chi.URLParam(r, "id")
	userID := middleware.GetUserIDFromContext(r.Context())

	if err := h.service.UnmuteChat(chatID, userID); err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to unmute chat"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) PinChat(w http.ResponseWriter, r *http.Request) {
	chatID := chi.URLParam(r, "id")
	userID := middleware.GetUserIDFromContext(r.Context())

	if err := h.service.PinChat(chatID, userID); err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to pin chat"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) UnpinChat(w http.ResponseWriter, r *http.Request) {
	chatID := chi.URLParam(r, "id")
	userID := middleware.GetUserIDFromContext(r.Context())

	if err := h.service.UnpinChat(chatID, userID); err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to unpin chat"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

type pinMsgReq struct {
	MessageID string `json:"messageId"`
}

func (h *Handler) PinMessage(w http.ResponseWriter, r *http.Request) {
	chatID := chi.URLParam(r, "id")
	userID := middleware.GetUserIDFromContext(r.Context())

	var req pinMsgReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MessageID == "" {
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"messageId required"}}`, http.StatusBadRequest)
		return
	}

	msg, err := h.service.PinMessage(chatID, userID, req.MessageID)
	if err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to pin message"}}`, http.StatusInternalServerError)
		return
	}

	if h.hub != nil {
		h.hub.BroadcastToChat(chatID, "", "message:pinned", map[string]interface{}{
			"chat_id":    chatID,
			"message_id": msg.ID,
			"content":    msg.Content,
			"pinned_by":  userID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}

func (h *Handler) UnpinMessage(w http.ResponseWriter, r *http.Request) {
	chatID := chi.URLParam(r, "id")
	userID := middleware.GetUserIDFromContext(r.Context())

	if err := h.service.UnpinMessage(chatID, userID); err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to unpin message"}}`, http.StatusInternalServerError)
		return
	}

	if h.hub != nil {
		h.hub.BroadcastToChat(chatID, "", "message:unpinned", map[string]interface{}{
			"chat_id":     chatID,
			"unpinned_by": userID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
