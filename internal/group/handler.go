package group

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

type createGroupReq struct {
	Name string `json:"name"`
	Type string `json:"type"` // group or channel
}

type addMemberReq struct {
	UserID string `json:"user_id"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	var req createGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"invalid json"}}`, http.StatusBadRequest)
		return
	}

	chat, err := h.service.CreateGroup(userID, req.Name, req.Type)
	if err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to create group"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(chat)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	chatID := chi.URLParam(r, "id")
	userID := middleware.GetUserIDFromContext(r.Context())
	var req addMemberReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"invalid json"}}`, http.StatusBadRequest)
		return
	}

	if err := h.service.AddMember(chatID, userID, req.UserID); err != nil {
		http.Error(w, `{"error":{"code":"FORBIDDEN","message":"`+err.Error()+`"}}`, http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	chatID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "userId")
	userID := middleware.GetUserIDFromContext(r.Context())

	if err := h.service.RemoveMember(chatID, userID, targetUserID); err != nil {
		http.Error(w, `{"error":{"code":"FORBIDDEN","message":"`+err.Error()+`"}}`, http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
