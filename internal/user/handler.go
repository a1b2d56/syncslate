package user

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

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	userID := middleware.GetUserIDFromContext(r.Context())

	profiles, err := h.service.Search(q, userID)
	if err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to search users"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "userId")
	userID := middleware.GetUserIDFromContext(r.Context())

	profile, err := h.service.GetByID(targetID, userID)
	if err != nil {
		http.Error(w, `{"error":{"code":"NOT_FOUND","message":"user not found"}}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	var req UpdateProfileReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"invalid json"}}`, http.StatusBadRequest)
		return
	}

	user, err := h.service.UpdateProfile(userID, req)
	if err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to update profile"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
