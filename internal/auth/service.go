package auth

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"syncslate/internal/config"
	"syncslate/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameTaken   = errors.New("username already taken")
	ErrInvalidCreds    = errors.New("invalid username or password")
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidToken    = errors.New("invalid or expired token")
)

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type Service struct {
	db     *sql.DB
	config *config.Config
}

func NewService(db *sql.DB, cfg *config.Config) *Service {
	return &Service{db: db, config: cfg}
}

type AuthResult struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         *models.User `json:"user"`
}

func (s *Service) Register(username, password, displayName string) (*AuthResult, error) {
	if len(username) < 3 || len(username) > 32 {
		return nil, errors.New("username must be between 3 and 32 characters")
	}
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = LOWER(?))`, username).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	userID := uuid.New().String()
	now := time.Now()

	_, err = s.db.Exec(`INSERT INTO users (id, username, password_hash, display_name, bio, discoverable, ghost_mode, created_at, updated_at)
		VALUES (?, ?, ?, ?, '', 1, 0, ?, ?)`,
		userID, username, string(hashedBytes), displayName, now.UnixMilli(), now.UnixMilli())
	if err != nil {
		return nil, err
	}

	// Auto-create "Saved Messages" chat
	savedChatID := uuid.New().String()
	_, _ = s.db.Exec(`INSERT INTO chats (id, type, created_at, updated_at) VALUES (?, 'saved', ?, ?)`, savedChatID, now.UnixMilli(), now.UnixMilli())
	_, _ = s.db.Exec(`INSERT INTO chat_members (chat_id, user_id, role, joined_at) VALUES (?, ?, 'owner', ?)`, savedChatID, userID, now.UnixMilli())

	user := &models.User{
		ID:           userID,
		Username:     username,
		DisplayName:  displayName,
		Bio:          "",
		Discoverable: true,
		GhostMode:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	return s.createSession(user)
}

func (s *Service) Login(username, password string) (*AuthResult, error) {
	var user models.User
	var passHash string
	var createdAtMs, updatedAtMs int64
	var avatarID sql.NullString
	var disc, ghost int

	err := s.db.QueryRow(`SELECT id, username, password_hash, display_name, bio, avatar_media_id, discoverable, ghost_mode, created_at, updated_at
		FROM users WHERE LOWER(username) = LOWER(?)`, username).
		Scan(&user.ID, &user.Username, &passHash, &user.DisplayName, &user.Bio, &avatarID, &disc, &ghost, &createdAtMs, &updatedAtMs)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidCreds
	} else if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passHash), []byte(password)); err != nil {
		return nil, ErrInvalidCreds
	}

	if avatarID.Valid {
		user.AvatarMediaID = &avatarID.String
	}
	user.Discoverable = disc == 1
	user.GhostMode = ghost == 1
	user.CreatedAt = time.UnixMilli(createdAtMs)
	user.UpdatedAt = time.UnixMilli(updatedAtMs)

	return s.createSession(&user)
}

func (s *Service) createSession(user *models.User) (*AuthResult, error) {
	now := time.Now()

	// Access token
	accessClaims := &Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.JWTAccessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, err
	}

	// Refresh token
	refreshToken := uuid.New().String() + "-" + uuid.New().String()
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHashStr := hex.EncodeToString(hash[:])

	sessionID := uuid.New().String()
	expiresAt := now.Add(s.config.JWTRefreshTTL)

	_, err = s.db.Exec(`INSERT INTO sessions (id, user_id, token_hash, device_info, created_at, expires_at) VALUES (?, ?, ?, 'mobile', ?, ?)`,
		sessionID, user.ID, tokenHashStr, now.UnixMilli(), expiresAt.UnixMilli())
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *Service) ValidateToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || claims.UserID == "" {
		return "", ErrInvalidToken
	}

	return claims.UserID, nil
}

func (s *Service) Refresh(refreshToken string) (*AuthResult, error) {
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHashStr := hex.EncodeToString(hash[:])

	var sessionID, userID string
	var expiresAtMs int64
	err := s.db.QueryRow(`SELECT id, user_id, expires_at FROM sessions WHERE token_hash = ?`, tokenHashStr).Scan(&sessionID, &userID, &expiresAtMs)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if time.Now().UnixMilli() > expiresAtMs {
		s.db.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)
		return nil, ErrInvalidToken
	}

	// Delete old session
	s.db.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)

	var user models.User
	var disc, ghost int
	var createdAtMs, updatedAtMs int64
	var avatarID sql.NullString

	err = s.db.QueryRow(`SELECT id, username, display_name, bio, avatar_media_id, discoverable, ghost_mode, created_at, updated_at FROM users WHERE id = ?`, userID).
		Scan(&user.ID, &user.Username, &user.DisplayName, &user.Bio, &avatarID, &disc, &ghost, &createdAtMs, &updatedAtMs)
	if err != nil {
		return nil, err
	}

	if avatarID.Valid {
		user.AvatarMediaID = &avatarID.String
	}
	user.Discoverable = disc == 1
	user.GhostMode = ghost == 1

	return s.createSession(&user)
}

func (s *Service) Logout(userID string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

func (s *Service) GetUserByID(userID string) (*models.User, error) {
	var user models.User
	var disc, ghost int
	var createdAtMs, updatedAtMs int64
	var avatarID sql.NullString

	err := s.db.QueryRow(`SELECT id, username, display_name, bio, avatar_media_id, discoverable, ghost_mode, created_at, updated_at FROM users WHERE id = ?`, userID).
		Scan(&user.ID, &user.Username, &user.DisplayName, &user.Bio, &avatarID, &disc, &ghost, &createdAtMs, &updatedAtMs)
	if err != nil {
		return nil, err
	}

	if avatarID.Valid {
		user.AvatarMediaID = &avatarID.String
	}
	user.Discoverable = disc == 1
	user.GhostMode = ghost == 1
	user.CreatedAt = time.UnixMilli(createdAtMs)
	user.UpdatedAt = time.UnixMilli(updatedAtMs)
	return &user, nil
}
