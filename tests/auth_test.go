package tests

import (
	"os"
	"testing"

	"syncslate/internal/auth"
	"syncslate/internal/config"
	"syncslate/internal/database"
)

func TestAuthRegisterLogin(t *testing.T) {
	dbPath := "./test_auth.db"
	defer os.Remove(dbPath)

	db, err := database.New(dbPath, 2)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	if err := db.RunMigrations("../migrations"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	cfg := config.Load()
	service := auth.NewService(db.DB, cfg)

	// 1. Register
	res, err := service.Register("alice", "password123", "Alice")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Errorf("expected tokens to be non-empty")
	}
	if res.User.Username != "alice" {
		t.Errorf("expected username alice, got %s", res.User.Username)
	}

	// 2. Duplicate username
	_, err = service.Register("alice", "anotherpass", "Alice Two")
	if err == nil {
		t.Errorf("expected duplicate username error, got nil")
	}

	// 3. Login
	loginRes, err := service.Login("alice", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loginRes.User.ID != res.User.ID {
		t.Errorf("user ID mismatch")
	}

	// 4. Invalid credentials
	_, err = service.Login("alice", "wrongpassword")
	if err == nil {
		t.Errorf("expected error for wrong password")
	}
}
