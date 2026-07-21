package group

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

func (s *Service) CreateGroup(creatorID, name, groupType string) (*models.Chat, error) {
	if groupType != "group" && groupType != "channel" {
		groupType = "group"
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}

	nowMs := time.Now().UnixMilli()
	chatID := uuid.New().String()

	_, err = tx.Exec(`INSERT INTO chats (id, type, created_at, updated_at) VALUES (?, ?, ?, ?)`, chatID, groupType, nowMs, nowMs)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	_, err = tx.Exec(`INSERT INTO chat_members (chat_id, user_id, role, joined_at) VALUES (?, ?, 'owner', ?)`, chatID, creatorID, nowMs)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.Chat{
		ID:        chatID,
		Type:      groupType,
		Name:      &name,
		CreatedAt: time.UnixMilli(nowMs),
		UpdatedAt: time.UnixMilli(nowMs),
	}, nil
}

func (s *Service) AddMember(chatID, requesterID, targetUserID string) error {
	var role string
	err := s.db.QueryRow(`SELECT role FROM chat_members WHERE chat_id = ? AND user_id = ?`, chatID, requesterID).Scan(&role)
	if err != nil || (role != "owner" && role != "admin") {
		return errors.New("unauthorized to add members")
	}

	nowMs := time.Now().UnixMilli()
	_, err = s.db.Exec(`INSERT OR IGNORE INTO chat_members (chat_id, user_id, role, joined_at) VALUES (?, ?, 'member', ?)`, chatID, targetUserID, nowMs)
	return err
}

func (s *Service) RemoveMember(chatID, requesterID, targetUserID string) error {
	if requesterID != targetUserID {
		var role string
		err := s.db.QueryRow(`SELECT role FROM chat_members WHERE chat_id = ? AND user_id = ?`, chatID, requesterID).Scan(&role)
		if err != nil || (role != "owner" && role != "admin") {
			return errors.New("unauthorized to remove member")
		}
	}

	_, err := s.db.Exec(`DELETE FROM chat_members WHERE chat_id = ? AND user_id = ?`, chatID, targetUserID)
	return err
}
