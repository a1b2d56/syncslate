package tests

import (
	"os"
	"testing"
	"time"

	"syncslate/internal/message"
	"syncslate/internal/database"
)

func TestMessageEditWindowLimit(t *testing.T) {
	dbPath := "./test_msg.db"
	defer os.Remove(dbPath)

	db, err := database.New(dbPath, 2)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	if err := db.RunMigrations("../migrations"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	msgService := message.NewService(db.DB)

	// Setup mock user & chat
	userID := "user-1"
	chatID := "chat-1"
	db.Exec(`INSERT INTO users (id, username, password_hash, created_at, updated_at) VALUES ('user-1', 'u1', 'hash', 0, 0)`)
	db.Exec(`INSERT INTO chats (id, type, created_at, updated_at) VALUES ('chat-1', 'direct', 0, 0)`)
	db.Exec(`INSERT INTO chat_members (chat_id, user_id, role, joined_at) VALUES ('chat-1', 'user-1', 'member', 0)`)

	// 1. Create message
	msg, err := msgService.Create(chatID, userID, "Hello World", nil, nil)
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	// 2. Edit within window -> success
	editedMsg, err := msgService.EditMessage(msg.ID, userID, "Hello Updated")
	if err != nil {
		t.Fatalf("edit within window failed: %v", err)
	}
	if editedMsg.Content != "Hello Updated" {
		t.Errorf("content not updated")
	}

	// 3. Backdate message to 50 hours ago to test 48-hour edit window expiration
	fiftyHoursAgoMs := time.Now().Add(-50 * time.Hour).UnixMilli()
	db.Exec(`UPDATE messages SET created_at = ? WHERE id = ?`, fiftyHoursAgoMs, msg.ID)

	_, err = msgService.EditMessage(msg.ID, userID, "Should Fail")
	if err != message.ErrEditWindowExpired {
		t.Errorf("expected ErrEditWindowExpired, got %v", err)
	}
}
