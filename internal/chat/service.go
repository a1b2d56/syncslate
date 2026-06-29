package chat

import (
	"database/sql"
	"time"

	"syncslate/internal/models"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetUserChats(userID string) ([]models.Chat, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.type, c.created_at, c.updated_at
		FROM chats c
		JOIN chat_members cm ON c.id = cm.chat_id
		WHERE cm.user_id = ?
		ORDER BY c.updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		var c models.Chat
		var createdAtMs, updatedAtMs int64
		if err := rows.Scan(&c.ID, &c.Type, &createdAtMs, &updatedAtMs); err != nil {
			return nil, err
		}

		c.CreatedAt = time.UnixMilli(createdAtMs)
		c.UpdatedAt = time.UnixMilli(updatedAtMs)

		// Peer profile if direct chat
		if c.Type == "direct" {
			var peer models.UserProfile
			var avatarID sql.NullString
			var disc, ghost int
			err := s.db.QueryRow(`
				SELECT u.id, u.username, u.display_name, u.bio, u.avatar_media_id, u.discoverable, u.ghost_mode
				FROM users u
				JOIN chat_members cm ON u.id = cm.user_id
				WHERE cm.chat_id = ? AND cm.user_id != ?`, c.ID, userID).
				Scan(&peer.ID, &peer.Username, &peer.DisplayName, &peer.Bio, &avatarID, &disc, &ghost)

			if err == nil {
				if avatarID.Valid {
					peer.AvatarMediaID = &avatarID.String
				}
				peer.Discoverable = disc == 1
				peer.GhostMode = ghost == 1
				peer.Status = "offline"
				c.Peer = &peer
			}
		}

		// Last message
		var lastMsg models.Message
		var msgCreatedAtMs int64
		var mediaID, replyToID sql.NullString
		var isEdited, isDeleted int
		err = s.db.QueryRow(`
			SELECT id, sender_id, content, media_id, reply_to_id, is_edited, is_deleted, created_at
			FROM messages
			WHERE chat_id = ?
			ORDER BY created_at DESC LIMIT 1`, c.ID).
			Scan(&lastMsg.ID, &lastMsg.SenderID, &lastMsg.Content, &mediaID, &replyToID, &isEdited, &isDeleted, &msgCreatedAtMs)

		if err == nil {
			lastMsg.ChatID = c.ID
			if mediaID.Valid {
				lastMsg.MediaID = &mediaID.String
			}
			if replyToID.Valid {
				lastMsg.ReplyToID = &replyToID.String
			}
			lastMsg.IsEdited = isEdited == 1
			lastMsg.IsDeleted = isDeleted == 1
			lastMsg.CreatedAt = time.UnixMilli(msgCreatedAtMs)
			c.LastMessage = &lastMsg
		}

		// Unread count
		var lastReadID string
		_ = s.db.QueryRow(`SELECT last_read_message_id FROM read_receipts WHERE chat_id = ? AND user_id = ?`, c.ID, userID).Scan(&lastReadID)

		if lastReadID != "" {
			_ = s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE chat_id = ? AND id > ? AND sender_id != ?`, c.ID, lastReadID, userID).Scan(&c.UnreadCount)
		} else {
			_ = s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE chat_id = ? AND sender_id != ?`, c.ID, userID).Scan(&c.UnreadCount)
		}

		chats = append(chats, c)
	}

	if chats == nil {
		chats = []models.Chat{}
	}
	return chats, nil
}

func (s *Service) GetChatMessages(chatID, userID, before string, limit int) ([]models.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var query string
	var args []interface{}

	if before != "" {
		var beforeTimeMs int64
		_ = s.db.QueryRow(`SELECT created_at FROM messages WHERE id = ?`, before).Scan(&beforeTimeMs)
		if beforeTimeMs > 0 {
			query = `SELECT id, chat_id, sender_id, content, media_id, reply_to_id, is_edited, is_deleted, created_at
				FROM messages WHERE chat_id = ? AND created_at < ? ORDER BY created_at DESC LIMIT ?`
			args = []interface{}{chatID, beforeTimeMs, limit}
		}
	}

	if query == "" {
		query = `SELECT id, chat_id, sender_id, content, media_id, reply_to_id, is_edited, is_deleted, created_at
			FROM messages WHERE chat_id = ? ORDER BY created_at DESC LIMIT ?`
		args = []interface{}{chatID, limit}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		var createdAtMs int64
		var mediaID, replyToID sql.NullString
		var isEdited, isDeleted int

		if err := rows.Scan(&m.ID, &m.ChatID, &m.SenderID, &m.Content, &mediaID, &replyToID, &isEdited, &isDeleted, &createdAtMs); err != nil {
			return nil, err
		}

		if mediaID.Valid {
			m.MediaID = &mediaID.String
		}
		if replyToID.Valid {
			m.ReplyToID = &replyToID.String
		}
		m.IsEdited = isEdited == 1
		m.IsDeleted = isDeleted == 1
		m.CreatedAt = time.UnixMilli(createdAtMs)

		messages = append(messages, m)
	}

	if messages == nil {
		messages = []models.Message{}
	}
	return messages, nil
}
