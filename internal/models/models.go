package models

import "time"

type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	PasswordHash  string    `json:"-"`
	DisplayName   string    `json:"display_name"`
	Bio           string    `json:"bio"`
	AvatarMediaID *string   `json:"avatar_media_id,omitempty"`
	Discoverable  bool      `json:"discoverable"`
	GhostMode     bool      `json:"ghost_mode"`
	LastSeen      *int64    `json:"last_seen,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type UserProfile struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	DisplayName   string  `json:"display_name"`
	Bio           string  `json:"bio"`
	AvatarMediaID *string `json:"avatar_media_id,omitempty"`
	Discoverable  bool    `json:"discoverable"`
	GhostMode     bool    `json:"ghost_mode"`
	Status        string  `json:"status"`
	LastSeen      *int64  `json:"lastSeen,omitempty"`
	IsOnline      bool    `json:"isOnline"`
}

type Session struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	TokenHash  string    `json:"-"`
	DeviceInfo string    `json:"device_info"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type Contact struct {
	ID        string      `json:"id"`
	OwnerID   string      `json:"owner_id"`
	ContactID string      `json:"contactId"`
	User      UserProfile `json:"user"`
	Profile   UserProfile `json:"profile"`
	CreatedAt time.Time   `json:"created_at"`
}

type MessageRequest struct {
	ID             string      `json:"id"`
	SenderID       string      `json:"sender_id"`
	RecipientID    string      `json:"recipient_id"`
	Sender         UserProfile `json:"sender"`
	InitialMessage string      `json:"initial_message"`
	Status         string      `json:"status"` // pending, accepted, declined, blocked
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type Chat struct {
	ID                   string       `json:"id"`
	Type                 string       `json:"type"` // direct, group, channel, saved
	Name                 *string      `json:"name,omitempty"`
	Peer                 *UserProfile `json:"peer,omitempty"`
	LastMessage          *Message     `json:"last_message,omitempty"`
	UnreadCount          int          `json:"unread_count"`
	IsMuted              bool         `json:"isMuted"`
	MuteUntil            *int64       `json:"muteUntil,omitempty"`
	MemberCount          int          `json:"memberCount"`
	OnlineCount          int          `json:"onlineCount"`
	PinnedMessageID      *string      `json:"pinnedMessageId,omitempty"`
	PinnedMessageContent *string      `json:"pinnedMessageContent,omitempty"`
	IsPinned             bool         `json:"isPinned"`
	PinnedOrder          int          `json:"pinnedOrder"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

type ChatMember struct {
	ChatID   string `json:"chat_id"`
	UserID   string `json:"user_id"`
	Role     string `json:"role"` // owner, admin, member
	JoinedAt int64  `json:"joined_at"`
}

type Message struct {
	ID          string    `json:"id"`
	ChatID      string    `json:"chat_id"`
	SenderID    string    `json:"sender_id"`
	Content     string    `json:"content"`
	MediaID     *string   `json:"media_id,omitempty"`
	ReplyToID   *string   `json:"reply_to_id,omitempty"`
	IsEdited    bool      `json:"is_edited"`
	EditedAt    *int64    `json:"edited_at,omitempty"`
	IsDeleted   bool      `json:"is_deleted"`
	DeletedAt   *int64    `json:"deleted_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type ReadReceipt struct {
	ChatID            string `json:"chat_id"`
	UserID            string `json:"user_id"`
	LastReadMessageID string `json:"last_read_message_id"`
	UpdatedAt         int64  `json:"updated_at"`
}

type Media struct {
	ID         string    `json:"id"`
	UploaderID string    `json:"uploader_id"`
	FileName   string    `json:"file_name"`
	FileType   string    `json:"file_type"`
	FileSize   int64     `json:"file_size"`
	FilePath   string    `json:"file_path"`
	CreatedAt  time.Time `json:"created_at"`
}

type Block struct {
	BlockerID string    `json:"blocker_id"`
	BlockedID string    `json:"blocked_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatFolder struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Name        string   `json:"name"`
	Icon        string   `json:"icon"`
	FilterFlags int      `json:"filterFlags"`
	FolderOrder int      `json:"folderOrder"`
	ChatIDs     []string `json:"chatIds"`
	CreatedAt   int64    `json:"createdAt"`
	UpdatedAt   int64    `json:"updatedAt"`
}
