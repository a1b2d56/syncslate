package user

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

func (s *Service) Search(query, currentUserID string) ([]models.UserProfile, error) {
	rows, err := s.db.Query(`
		SELECT id, username, display_name, bio, avatar_media_id, discoverable, ghost_mode
		FROM users
		WHERE discoverable = 1
		  AND LOWER(username) LIKE LOWER(?)
		  AND id != ?
		  AND id NOT IN (SELECT blocked_id FROM blocks WHERE blocker_id = ?)
		  AND id NOT IN (SELECT blocker_id FROM blocks WHERE blocked_id = ?)
		LIMIT 20`, "%"+query+"%", currentUserID, currentUserID, currentUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []models.UserProfile
	for rows.Next() {
		var p models.UserProfile
		var avatarID sql.NullString
		var disc, ghost int
		if err := rows.Scan(&p.ID, &p.Username, &p.DisplayName, &p.Bio, &avatarID, &disc, &ghost); err != nil {
			return nil, err
		}
		if avatarID.Valid {
			p.AvatarMediaID = &avatarID.String
		}
		p.Discoverable = disc == 1
		p.GhostMode = ghost == 1
		p.Status = "offline"
		profiles = append(profiles, p)
	}

	if profiles == nil {
		profiles = []models.UserProfile{}
	}
	return profiles, nil
}

func (s *Service) GetByID(userID, requesterID string) (*models.UserProfile, error) {
	// Block check
	var isBlocked bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM blocks WHERE (blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?))`,
		userID, requesterID, requesterID, userID).Scan(&isBlocked)
	if err != nil {
		return nil, err
	}
	if isBlocked {
		return nil, sql.ErrNoRows
	}

	var p models.UserProfile
	var avatarID sql.NullString
	var disc, ghost int
	err = s.db.QueryRow(`SELECT id, username, display_name, bio, avatar_media_id, discoverable, ghost_mode FROM users WHERE id = ?`, userID).
		Scan(&p.ID, &p.Username, &p.DisplayName, &p.Bio, &avatarID, &disc, &ghost)
	if err != nil {
		return nil, err
	}

	if avatarID.Valid {
		p.AvatarMediaID = &avatarID.String
	}
	p.Discoverable = disc == 1
	p.GhostMode = ghost == 1
	p.Status = "offline"

	return &p, nil
}

type UpdateProfileReq struct {
	DisplayName  *string `json:"display_name,omitempty"`
	Bio          *string `json:"bio,omitempty"`
	AvatarID     *string `json:"avatar_media_id,omitempty"`
	Discoverable *bool   `json:"discoverable,omitempty"`
	GhostMode    *bool   `json:"ghost_mode,omitempty"`
}

func (s *Service) UpdateProfile(userID string, req UpdateProfileReq) (*models.User, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()

	if req.DisplayName != nil {
		_, _ = tx.Exec(`UPDATE users SET display_name = ?, updated_at = ? WHERE id = ?`, *req.DisplayName, now, userID)
	}
	if req.Bio != nil {
		_, _ = tx.Exec(`UPDATE users SET bio = ?, updated_at = ? WHERE id = ?`, *req.Bio, now, userID)
	}
	if req.AvatarID != nil {
		_, _ = tx.Exec(`UPDATE users SET avatar_media_id = ?, updated_at = ? WHERE id = ?`, *req.AvatarID, now, userID)
	}
	if req.Discoverable != nil {
		discVal := 0
		if *req.Discoverable {
			discVal = 1
		}
		_, _ = tx.Exec(`UPDATE users SET discoverable = ?, updated_at = ? WHERE id = ?`, discVal, now, userID)
	}
	if req.GhostMode != nil {
		ghostVal := 0
		if *req.GhostMode {
			ghostVal = 1
		}
		_, _ = tx.Exec(`UPDATE users SET ghost_mode = ?, updated_at = ? WHERE id = ?`, ghostVal, now, userID)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

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
	user.CreatedAt = time.UnixMilli(createdAtMs)
	user.UpdatedAt = time.UnixMilli(updatedAtMs)

	return &user, nil
}
