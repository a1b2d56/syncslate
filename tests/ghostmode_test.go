package tests

import (
	"os"
	"testing"

	"syncslate/internal/database"
	"syncslate/internal/user"
)

func TestGhostModeServerSideEnforcement(t *testing.T) {
	dbPath := "./test_ghost.db"
	defer os.Remove(dbPath)

	db, err := database.New(dbPath, 2)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	if err := db.RunMigrations("../migrations"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	userService := user.NewService(db.DB)

	db.Exec(`INSERT INTO users (id, username, password_hash, ghost_mode, created_at, updated_at) VALUES ('u1', 'ghostuser', 'hash', 1, 0, 0)`)

	ghostTrue := true
	updatedUser, err := userService.UpdateProfile("u1", user.UpdateProfileReq{GhostMode: &ghostTrue})
	if err != nil {
		t.Fatalf("update profile failed: %v", err)
	}

	if !updatedUser.GhostMode {
		t.Errorf("expected GhostMode to be true")
	}
}
