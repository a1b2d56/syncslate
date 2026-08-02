package request

import (
	"database/sql"
	"errors"
	"time"

	"syncslate/internal/models"

	"github.com/google/uuid"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetRequests(recipientID string) ([]models.MessageRequest, error) {
	rows, err := s.db.Query(`
		SELECT r.id, r.sender_id, r.recipient_id, r.initial_message, r.status, r.created_at, r.updated_at,
		       u.username, u.display_name, u.bio, u.avatar_media_id, u.discoverable, u.ghost_mode
		FROM message_requests r
		JOIN users u ON r.sender_id = u.id
		WHERE r.recipient_id = ? AND r.status = 'pending'
		ORDER BY r.created_at DESC`, recipientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reqs []models.MessageRequest
	for rows.Next() {
		var mr models.MessageRequest
		var createdAtMs, updatedAtMs int64
		var avatarID sql.NullString
		var disc, ghost int

		err := rows.Scan(&mr.ID, &mr.SenderID, &mr.RecipientID, &mr.InitialMessage, &mr.Status, &createdAtMs, &updatedAtMs,
			&mr.Sender.Username, &mr.Sender.DisplayName, &mr.Sender.Bio, &avatarID, &disc, &ghost)
		if err != nil {
			return nil, err
		}

		mr.Sender.ID = mr.SenderID
		if avatarID.Valid {
			mr.Sender.AvatarMediaID = &avatarID.String
		}
		mr.Sender.Discoverable = disc == 1
		mr.Sender.GhostMode = ghost == 1
		mr.Sender.Status = "offline"
		mr.CreatedAt = time.UnixMilli(createdAtMs)
		mr.UpdatedAt = time.UnixMilli(updatedAtMs)

		reqs = append(reqs, mr)
	}

	if reqs == nil {
		reqs = []models.MessageRequest{}
	}
	return reqs, nil
}

func (s *Service) AcceptRequest(requestID, recipientID string) (*models.Chat, error) {
	var senderID, status string
	var initialMsg string
	err := s.db.QueryRow(`SELECT sender_id, initial_message, status FROM message_requests WHERE id = ? AND recipient_id = ?`, requestID, recipientID).
		Scan(&senderID, &initialMsg, &status)
	if err != nil {
		return nil, errors.New("request not found")
	}

	if status != "pending" {
		return nil, errors.New("request already processed")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}

	nowMs := time.Now().UnixMilli()

	// Update request status
	_, err = tx.Exec(`UPDATE message_requests SET status = 'accepted', updated_at = ? WHERE id = ?`, nowMs, requestID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Create direct chat
	chatID := uuid.New().String()
	_, err = tx.Exec(`INSERT INTO chats (id, type, created_at, updated_at) VALUES (?, 'direct', ?, ?)`, chatID, nowMs, nowMs)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Add members
	_, err = tx.Exec(`INSERT INTO chat_members (chat_id, user_id, role, joined_at) VALUES (?, ?, 'member', ?), (?, ?, 'member', ?)`,
		chatID, senderID, nowMs, chatID, recipientID, nowMs)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Move initial message into chat if non-empty
	if initialMsg != "" {
		msgID := uuid.New().String()
		_, _ = tx.Exec(`INSERT INTO messages (id, chat_id, sender_id, content, is_edited, is_deleted, created_at) VALUES (?, ?, ?, ?, 0, 0, ?)`,
			msgID, chatID, senderID, initialMsg, nowMs)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	chat := &models.Chat{
		ID:        chatID,
		Type:      "direct",
		CreatedAt: time.UnixMilli(nowMs),
		UpdatedAt: time.UnixMilli(nowMs),
	}

	return chat, nil
}

func (s *Service) CreateRequest(senderID, recipientID, initialMessage string) (*models.MessageRequest, error) {
	if recipientID == "" {
		return nil, errors.New("recipient_id is required")
	}

	if senderID == recipientID {
		return nil, errors.New("cannot send message request to yourself")
	}

	var recipientExists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)`, recipientID).Scan(&recipientExists)
	if err != nil || !recipientExists {
		return nil, errors.New("recipient not found")
	}

	// Check if blocked
	var isBlocked bool
	_ = s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM blocks WHERE (blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?))`,
		senderID, recipientID, recipientID, senderID).Scan(&isBlocked)
	if isBlocked {
		return nil, errors.New("cannot send message request to this user")
	}

	reqID := uuid.New().String()
	nowMs := time.Now().UnixMilli()

	_, err = s.db.Exec(`INSERT INTO message_requests (id, sender_id, recipient_id, initial_message, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'pending', ?, ?)
		ON CONFLICT(sender_id, recipient_id) DO UPDATE SET initial_message = excluded.initial_message, status = 'pending', updated_at = excluded.updated_at`,
		reqID, senderID, recipientID, initialMessage, nowMs, nowMs)
	if err != nil {
		return nil, err
	}

	var mr models.MessageRequest
	var createdAtMs, updatedAtMs int64
	var avatarID sql.NullString
	var disc, ghost int

	err = s.db.QueryRow(`
		SELECT r.id, r.sender_id, r.recipient_id, r.initial_message, r.status, r.created_at, r.updated_at,
		       u.username, u.display_name, u.bio, u.avatar_media_id, u.discoverable, u.ghost_mode
		FROM message_requests r
		JOIN users u ON r.sender_id = u.id
		WHERE r.sender_id = ? AND r.recipient_id = ?`, senderID, recipientID).
		Scan(&mr.ID, &mr.SenderID, &mr.RecipientID, &mr.InitialMessage, &mr.Status, &createdAtMs, &updatedAtMs,
			&mr.Sender.Username, &mr.Sender.DisplayName, &mr.Sender.Bio, &avatarID, &disc, &ghost)
	if err != nil {
		return nil, err
	}

	mr.Sender.ID = mr.SenderID
	if avatarID.Valid {
		mr.Sender.AvatarMediaID = &avatarID.String
	}
	mr.Sender.Discoverable = disc == 1
	mr.Sender.GhostMode = ghost == 1
	mr.Sender.Status = "offline"
	mr.CreatedAt = time.UnixMilli(createdAtMs)
	mr.UpdatedAt = time.UnixMilli(updatedAtMs)

	return &mr, nil
}

func (s *Service) DeclineRequest(requestID, recipientID string, block bool) error {
	var senderID string
	err := s.db.QueryRow(`SELECT sender_id FROM message_requests WHERE id = ? AND recipient_id = ?`, requestID, recipientID).Scan(&senderID)
	if err != nil {
		return errors.New("request not found")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	nowMs := time.Now().UnixMilli()

	newStatus := "declined"
	if block {
		newStatus = "blocked"
		_, _ = tx.Exec(`INSERT OR IGNORE INTO blocks (blocker_id, blocked_id, created_at) VALUES (?, ?, ?)`, recipientID, senderID, nowMs)
	}

	_, err = tx.Exec(`UPDATE message_requests SET status = ?, updated_at = ? WHERE id = ?`, newStatus, nowMs, requestID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
