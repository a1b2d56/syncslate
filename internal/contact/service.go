package contact

import (
	"database/sql"
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

func (s *Service) GetContacts(ownerID string) ([]models.Contact, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.contact_id, c.created_at, u.username, u.display_name, u.bio, u.avatar_media_id, u.discoverable, u.ghost_mode
		FROM contacts c
		JOIN users u ON c.contact_id = u.id
		WHERE c.owner_id = ?
		ORDER BY u.display_name ASC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var c models.Contact
		var createdAtMs int64
		var avatarID sql.NullString
		var disc, ghost int

		err := rows.Scan(&c.ID, &c.ContactID, &createdAtMs, &c.Profile.Username, &c.Profile.DisplayName, &c.Profile.Bio, &avatarID, &disc, &ghost)
		if err != nil {
			return nil, err
		}

		c.OwnerID = ownerID
		c.Profile.ID = c.ContactID
		if avatarID.Valid {
			c.Profile.AvatarMediaID = &avatarID.String
		}
		c.Profile.Discoverable = disc == 1
		c.Profile.GhostMode = ghost == 1
		c.Profile.Status = "offline"
		c.CreatedAt = time.UnixMilli(createdAtMs)

		contacts = append(contacts, c)
	}

	if contacts == nil {
		contacts = []models.Contact{}
	}
	return contacts, nil
}

func (s *Service) AddContact(ownerID, contactID string) (*models.Contact, error) {
	id := uuid.New().String()
	now := time.Now()
	nowMs := now.UnixMilli()

	_, err := s.db.Exec(`INSERT OR IGNORE INTO contacts (id, owner_id, contact_id, created_at) VALUES (?, ?, ?, ?)`,
		id, ownerID, contactID, nowMs)
	if err != nil {
		return nil, err
	}

	var c models.Contact
	c.ID = id
	c.OwnerID = ownerID
	c.ContactID = contactID
	c.CreatedAt = now

	return &c, nil
}

func (s *Service) RemoveContact(ownerID, contactID string) error {
	_, err := s.db.Exec(`DELETE FROM contacts WHERE owner_id = ? AND (contact_id = ? OR id = ?)`, ownerID, contactID, contactID)
	return err
}
