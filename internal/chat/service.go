package chat

import (
	"database/sql"
	"errors"
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
		SELECT c.id, c.type, c.created_at, c.updated_at, c.pinned_message_id,
		       COALESCE(cp.pinned_order, -1) as pinned_order,
		       cmute.mute_until,
		       CASE WHEN cmute.chat_id IS NOT NULL THEN 1 ELSE 0 END as is_muted
		FROM chats c
		JOIN chat_members cm ON c.id = cm.chat_id
		LEFT JOIN chat_pins cp ON c.id = cp.chat_id AND cp.user_id = ?
		LEFT JOIN chat_mutes cmute ON c.id = cmute.chat_id AND cmute.user_id = ?
		WHERE cm.user_id = ?
		ORDER BY (CASE WHEN cp.chat_id IS NOT NULL THEN 0 ELSE 1 END) ASC,
		         cp.pinned_order ASC,
		         c.updated_at DESC`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		var c models.Chat
		var createdAtMs, updatedAtMs int64
		var pinnedMsgID sql.NullString
		var pinnedOrder int
		var muteUntil sql.NullInt64
		var isMutedInt int

		if err := rows.Scan(&c.ID, &c.Type, &createdAtMs, &updatedAtMs, &pinnedMsgID, &pinnedOrder, &muteUntil, &isMutedInt); err != nil {
			return nil, err
		}

		c.CreatedAt = time.UnixMilli(createdAtMs)
		c.UpdatedAt = time.UnixMilli(updatedAtMs)
		if pinnedMsgID.Valid {
			c.PinnedMessageID = &pinnedMsgID.String
			// Fetch pinned message content
			var pContent string
			if err := s.db.QueryRow(`SELECT content FROM messages WHERE id = ?`, pinnedMsgID.String).Scan(&pContent); err == nil {
				c.PinnedMessageContent = &pContent
			}
		}

		if pinnedOrder >= 0 {
			c.IsPinned = true
			c.PinnedOrder = pinnedOrder
		}

		if isMutedInt == 1 {
			c.IsMuted = true
			if muteUntil.Valid {
				c.MuteUntil = &muteUntil.Int64
			}
		}

		// Member counts & online counts
		var memberCount int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM chat_members WHERE chat_id = ?`, c.ID).Scan(&memberCount)
		c.MemberCount = memberCount
		c.OnlineCount = 0 // Online count placeholder (or populated if clients connect)

		// Peer profile if direct chat
		if c.Type == "direct" {
			var peer models.UserProfile
			var avatarID sql.NullString
			var disc, ghost int
			var lastSeen sql.NullInt64
			err := s.db.QueryRow(`
				SELECT u.id, u.username, u.display_name, u.bio, u.avatar_media_id, u.discoverable, u.ghost_mode, u.last_seen
				FROM users u
				JOIN chat_members cm ON u.id = cm.user_id
				WHERE cm.chat_id = ? AND cm.user_id != ?`, c.ID, userID).
				Scan(&peer.ID, &peer.Username, &peer.DisplayName, &peer.Bio, &avatarID, &disc, &ghost, &lastSeen)

			if err == nil {
				if avatarID.Valid {
					peer.AvatarMediaID = &avatarID.String
				}
				peer.Discoverable = disc == 1
				peer.GhostMode = ghost == 1
				peer.Status = "offline"
				if lastSeen.Valid {
					peer.LastSeen = &lastSeen.Int64
				}
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

func (s *Service) MuteChat(chatID, userID string, muteUntil *int64) error {
	nowMs := time.Now().UnixMilli()
	if muteUntil != nil {
		_, err := s.db.Exec(`INSERT INTO chat_mutes (chat_id, user_id, mute_until, created_at) VALUES (?, ?, ?, ?)
			ON CONFLICT(chat_id, user_id) DO UPDATE SET mute_until = excluded.mute_until`, chatID, userID, *muteUntil, nowMs)
		return err
	}
	_, err := s.db.Exec(`INSERT INTO chat_mutes (chat_id, user_id, mute_until, created_at) VALUES (?, ?, NULL, ?)
		ON CONFLICT(chat_id, user_id) DO UPDATE SET mute_until = NULL`, chatID, userID, nowMs)
	return err
}

func (s *Service) UnmuteChat(chatID, userID string) error {
	_, err := s.db.Exec(`DELETE FROM chat_mutes WHERE chat_id = ? AND user_id = ?`, chatID, userID)
	return err
}

func (s *Service) PinChat(chatID, userID string) error {
	nowMs := time.Now().UnixMilli()
	var maxOrder sql.NullInt64
	_ = s.db.QueryRow(`SELECT MAX(pinned_order) FROM chat_pins WHERE user_id = ?`, userID).Scan(&maxOrder)
	nextOrder := 0
	if maxOrder.Valid {
		nextOrder = int(maxOrder.Int64) + 1
	}

	_, err := s.db.Exec(`INSERT INTO chat_pins (chat_id, user_id, pinned_order, created_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(chat_id, user_id) DO UPDATE SET pinned_order = excluded.pinned_order`, chatID, userID, nextOrder, nowMs)
	return err
}

func (s *Service) UnpinChat(chatID, userID string) error {
	_, err := s.db.Exec(`DELETE FROM chat_pins WHERE chat_id = ? AND user_id = ?`, chatID, userID)
	return err
}

func (s *Service) PinMessage(chatID, userID, messageID string) (*models.Message, error) {
	// Verify user is chat member
	var isMember bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM chat_members WHERE chat_id = ? AND user_id = ?)`, chatID, userID).Scan(&isMember)
	if err != nil || !isMember {
		return nil, errors.New("forbidden")
	}

	_, err = s.db.Exec(`UPDATE chats SET pinned_message_id = ? WHERE id = ?`, messageID, chatID)
	if err != nil {
		return nil, err
	}

	var m models.Message
	var createdAtMs int64
	var mediaID, replyToID sql.NullString
	var isEdited, isDeleted int
	err = s.db.QueryRow(`SELECT id, chat_id, sender_id, content, media_id, reply_to_id, is_edited, is_deleted, created_at FROM messages WHERE id = ? AND chat_id = ?`, messageID, chatID).
		Scan(&m.ID, &m.ChatID, &m.SenderID, &m.Content, &mediaID, &replyToID, &isEdited, &isDeleted, &createdAtMs)
	if err != nil {
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

	return &m, nil
}

func (s *Service) UnpinMessage(chatID, userID string) error {
	var isMember bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM chat_members WHERE chat_id = ? AND user_id = ?)`, chatID, userID).Scan(&isMember)
	if err != nil || !isMember {
		return errors.New("forbidden")
	}

	_, err = s.db.Exec(`UPDATE chats SET pinned_message_id = NULL WHERE id = ?`, chatID)
	return err
}
