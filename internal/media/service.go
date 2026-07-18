package media

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"syncslate/internal/models"

	"github.com/google/uuid"
)

var (
	ErrInvalidFileType = errors.New("unsupported file type")
	ErrFileTooLarge    = errors.New("file exceeds maximum allowed size")
)

var allowedMIMEs = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"video/mp4":       true,
	"audio/ogg":       true,
	"audio/mpeg":      true,
	"application/pdf": true,
}

type Service struct {
	db            *sql.DB
	store         Store
	maxUploadSize int64
}

func NewService(db *sql.DB, store Store, maxUploadSize int64) *Service {
	return &Service{db: db, store: store, maxUploadSize: maxUploadSize}
}

func (s *Service) Upload(uploaderID string, fileHeader *multipart.FileHeader) (*models.Media, error) {
	if fileHeader.Size > s.maxUploadSize {
		return nil, ErrFileTooLarge
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read first 512 bytes for MIME detection
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := httpDetectContentType(buf[:n], fileHeader.Header.Get("Content-Type"))

	if !allowedMIMEs[contentType] {
		return nil, ErrInvalidFileType
	}

	// Reset file reader
	_, _ = file.Seek(0, io.SeekStart)

	mediaID := uuid.New().String()
	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		ext = ".bin"
	}
	savedFileName := mediaID + ext

	relPath, err := s.store.Save(savedFileName, file)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	now := time.Now()
	nowMs := now.UnixMilli()

	_, err = s.db.Exec(`INSERT INTO media (id, uploader_id, file_name, file_type, file_size, file_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		mediaID, uploaderID, fileHeader.Filename, contentType, fileHeader.Size, relPath, nowMs)
	if err != nil {
		return nil, err
	}

	return &models.Media{
		ID:         mediaID,
		UploaderID: uploaderID,
		FileName:   fileHeader.Filename,
		FileType:   contentType,
		FileSize:   fileHeader.Size,
		FilePath:   relPath,
		CreatedAt:  now,
	}, nil
}

func (s *Service) Get(mediaID string) (*models.Media, io.ReadCloser, error) {
	var m models.Media
	var createdAtMs int64

	err := s.db.QueryRow(`SELECT id, uploader_id, file_name, file_type, file_size, file_path, created_at FROM media WHERE id = ?`, mediaID).
		Scan(&m.ID, &m.UploaderID, &m.FileName, &m.FileType, &m.FileSize, &m.FilePath, &createdAtMs)
	if err != nil {
		return nil, nil, err
	}

	m.CreatedAt = time.UnixMilli(createdAtMs)

	reader, err := s.store.Get(m.FilePath)
	if err != nil {
		return nil, nil, err
	}

	return &m, reader, nil
}

func httpDetectContentType(buf []byte, headerMIME string) string {
	if headerMIME != "" {
		parts := strings.Split(headerMIME, ";")
		mime := strings.TrimSpace(parts[0])
		if allowedMIMEs[mime] {
			return mime
		}
	}
	return "application/octet-stream"
}
