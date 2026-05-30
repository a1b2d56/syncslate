package auth

import (
	"encoding/json"
	"net/http"

	"syncslate/internal/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type registerReq struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"invalid json"}}`, http.StatusBadRequest)
		return
	}

	res, err := h.service.Register(req.Username, req.Password, req.DisplayName)
	if err != nil {
		if err == ErrUsernameTaken {
			http.Error(w, `{"error":{"code":"CONFLICT","message":"username taken"}}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"`+err.Error()+`"}}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"invalid json"}}`, http.StatusBadRequest)
		return
	}

	res, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"invalid credentials"}}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"invalid json"}}`, http.StatusBadRequest)
		return
	}

	res, err := h.service.Refresh(req.RefreshToken)
	if err != nil {
		http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"invalid refresh token"}}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	_ = h.service.Logout(userID)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	user, err := h.service.GetUserByID(userID)
	if err != nil {
		http.Error(w, `{"error":{"code":"NOT_FOUND","message":"user not found"}}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
