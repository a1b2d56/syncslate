package message

import (
	"encoding/json"
	"net/http"

	"syncslate/internal/middleware"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type createMsgReq struct {
	ChatID    string  `json:"chat_id"`
	Content   string  `json:"content"`
	MediaID   *string `json:"media_id,omitempty"`
	ReplyToID *string `json:"reply_to_id,omitempty"`
}

type editMsgReq struct {
	Content string `json:"content"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	var req createMsgReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"invalid json"}}`, http.StatusBadRequest)
		return
	}

	msg, err := h.service.Create(req.ChatID, userID, req.Content, req.MediaID, req.ReplyToID)
	if err != nil {
		http.Error(w, `{"error":{"code":"FORBIDDEN","message":"`+err.Error()+`"}}`, http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	msgID := chi.URLParam(r, "id")
	userID := middleware.GetUserIDFromContext(r.Context())
	var req editMsgReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"invalid json"}}`, http.StatusBadRequest)
		return
	}

	msg, err := h.service.EditMessage(msgID, userID, req.Content)
	if err != nil {
		if err == ErrEditWindowExpired {
			http.Error(w, `{"error":{"code":"FORBIDDEN","message":"edit window expired (48 hours)"}}`, http.StatusForbidden)
			return
		}
		if err == ErrUnauthorized {
			http.Error(w, `{"error":{"code":"FORBIDDEN","message":"unauthorized"}}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error":{"code":"NOT_FOUND","message":"message not found"}}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	msgID := chi.URLParam(r, "id")
	userID := middleware.GetUserIDFromContext(r.Context())
	forEveryone := r.URL.Query().Get("for_everyone") == "true"

	msg, err := h.service.DeleteMessage(msgID, userID, forEveryone)
	if err != nil {
		http.Error(w, `{"error":{"code":"FORBIDDEN","message":"`+err.Error()+`"}}`, http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}
