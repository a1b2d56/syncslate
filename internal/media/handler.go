package media

import (
	"encoding/json"
	"io"
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

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"failed to parse multipart form"}}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"missing file field"}}`, http.StatusBadRequest)
		return
	}
	file.Close()

	media, err := h.service.Upload(userID, header)
	if err != nil {
		if err == ErrInvalidFileType {
			http.Error(w, `{"error":{"code":"UNSUPPORTED_MEDIA_TYPE","message":"unsupported file type"}}`, http.StatusUnsupportedMediaType)
			return
		}
		if err == ErrFileTooLarge {
			http.Error(w, `{"error":{"code":"PAYLOAD_TOO_LARGE","message":"file exceeds maximum allowed size"}}`, http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to upload file"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"media_id":  media.ID,
		"url":       "/api/v1/media/" + media.ID,
		"file_type": media.FileType,
		"file_size": media.FileSize,
	})
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "mediaId")

	media, reader, err := h.service.Get(mediaID)
	if err != nil {
		http.Error(w, `{"error":{"code":"NOT_FOUND","message":"media not found"}}`, http.StatusNotFound)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", media.FileType)
	w.Header().Set("Content-Disposition", "inline; filename="+media.FileName)
	_, _ = io.Copy(w, reader)
}
