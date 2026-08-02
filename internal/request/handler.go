package request

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

type declineReq struct {
	Block bool `json:"block"`
}

type createReq struct {
	RecipientID       string `json:"recipient_id"`
	RecipientIDAlt    string `json:"recipientId"`
	InitialMessage    string `json:"initial_message"`
	InitialMessageAlt string `json:"initialMessage"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	reqs, err := h.service.GetRequests(userID)
	if err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to fetch requests"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reqs)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"invalid json body"}}`, http.StatusBadRequest)
		return
	}

	recipientID := req.RecipientID
	if recipientID == "" {
		recipientID = req.RecipientIDAlt
	}
	initialMessage := req.InitialMessage
	if initialMessage == "" {
		initialMessage = req.InitialMessageAlt
	}

	msgReq, err := h.service.CreateRequest(userID, recipientID, initialMessage)
	if err != nil {
		http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"`+err.Error()+`"}}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msgReq)
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	reqID := chi.URLParam(r, "id")

	chat, err := h.service.AcceptRequest(reqID, userID)
	if err != nil {
		http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"`+err.Error()+`"}}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

func (h *Handler) Decline(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	reqID := chi.URLParam(r, "id")
	var req declineReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	err := h.service.DeclineRequest(reqID, userID, req.Block)
	if err != nil {
		http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"`+err.Error()+`"}}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
