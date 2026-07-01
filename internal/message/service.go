package message

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"syncslate/internal/models"

	"github.com/google/uuid"
)

var (
	ErrEditWindowExpired = errors.New("message edit window expired (48 hours)")
	ErrUnauthorized      = errors.New("unauthorized to modify message")
	ErrMessageNotFound   = errors.New("message not found")
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(chatID, senderID, content string, mediaID, replyToID *string) (*models.Message, error) {
	// Verify chat membership
	var isMember bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM chat_members WHERE chat_id = ? AND user_id = ?)`, chatID, senderID).Scan(&isMember)
	if err != nil || !isMember {
		return nil, errors.New("user is not a member of this chat")
	}

	msgID := uuid.New().String()
	now := time.Now()
	nowMs := now.UnixMilli()

	_, err = s.db.Exec(`INSERT INTO messages (id, chat_id, sender_id, content, media_id, reply_to_id, is_edited, is_deleted, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, 0, ?)`,
		msgID, chatID, senderID, content, mediaID, replyToID, nowMs)
	if err != nil {
		return nil, fmt.Errorf("failed to insert message: %w", err)
	}

	// Update chat updated_at
	_, _ = s.db.Exec(`UPDATE chats SET updated_at = ? WHERE id = ?`, nowMs, chatID)

	msg := &models.Message{
		ID:        msgID,
		ChatID:    chatID,
		SenderID:  senderID,
		Content:   content,
		MediaID:   mediaID,
		ReplyToID: replyToID,
		IsEdited:  false,
		IsDeleted: false,
		CreatedAt: now,
	}

	return msg, nil
}

// EditMessage enforces strict 48-hour limit per project requirements!
func (s *Service) EditMessage(msgID, userID, newContent string) (*models.Message, error) {
	var senderID string
	var createdAtMs int64
	var isDeleted int

	err := s.db.QueryRow(`SELECT sender_id, created_at, is_deleted FROM messages WHERE id = ?`, msgID).
		Scan(&senderID, &createdAtMs, &isDeleted)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrMessageNotFound
	} else if err != nil {
		return nil, err
	}

	if senderID != userID {
		return nil, ErrUnauthorized
	}
	if isDeleted == 1 {
		return nil, errors.New("cannot edit deleted message")
	}

	createdAt := time.UnixMilli(createdAtMs)
	// Enforce 48 Hours Limit
	if time.Since(createdAt) > 48*time.Hour {
		return nil, ErrEditWindowExpired
	}

	nowMs := time.Now().UnixMilli()
	_, err = s.db.Exec(`UPDATE messages SET content = ?, is_edited = 1, edited_at = ? WHERE id = ?`, newContent, nowMs, msgID)
	if err != nil {
		return nil, err
	}

	var msg models.Message
	var chatID string
	var mediaID, replyToID sql.NullString

	err = s.db.QueryRow(`SELECT id, chat_id, sender_id, content, media_id, reply_to_id, created_at FROM messages WHERE id = ?`, msgID).
		Scan(&msg.ID, &chatID, &msg.SenderID, &msg.Content, &mediaID, &replyToID, &createdAtMs)
	if err != nil {
		return nil, err
	}

	msg.ChatID = chatID
	if mediaID.Valid {
		msg.MediaID = &mediaID.String
	}
	if replyToID.Valid {
		msg.ReplyToID = &replyToID.String
	}
	msg.IsEdited = true
	msg.EditedAt = &nowMs
	msg.CreatedAt = time.UnixMilli(createdAtMs)

	return &msg, nil
}

func (s *Service) DeleteMessage(msgID, userID string, forEveryone bool) (*models.Message, error) {
	var senderID, chatID string
	var createdAtMs int64

	err := s.db.QueryRow(`SELECT sender_id, chat_id, created_at FROM messages WHERE id = ?`, msgID).Scan(&senderID, &chatID, &createdAtMs)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrMessageNotFound
	} else if err != nil {
		return nil, err
	}

	if senderID != userID {
		return nil, ErrUnauthorized
	}

	nowMs := time.Now().UnixMilli()

	if forEveryone {
		_, err = s.db.Exec(`UPDATE messages SET is_deleted = 1, deleted_at = ? WHERE id = ?`, nowMs, msgID)
		if err != nil {
			return nil, err
		}
	}

	msg := &models.Message{
		ID:        msgID,
		ChatID:    chatID,
		SenderID:  senderID,
		IsDeleted: true,
		DeletedAt: &nowMs,
		CreatedAt: time.UnixMilli(createdAtMs),
	}

	return msg, nil
}
