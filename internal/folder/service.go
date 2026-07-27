package folder

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

func (s *Service) GetFolders(userID string) ([]models.ChatFolder, error) {
	rows, err := s.db.Query(`
		SELECT id, name, icon, filter_flags, folder_order, created_at, updated_at
		FROM chat_folders
		WHERE user_id = ?
		ORDER BY folder_order ASC, created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []models.ChatFolder
	for rows.Next() {
		var f models.ChatFolder
		if err := rows.Scan(&f.ID, &f.Name, &f.Icon, &f.FilterFlags, &f.FolderOrder, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		f.UserID = userID

		// Fetch explicitly added chat IDs
		cRows, err := s.db.Query(`SELECT chat_id FROM chat_folder_chats WHERE folder_id = ?`, f.ID)
		if err == nil {
			var chatIDs []string
			for cRows.Next() {
				var cid string
				if err := cRows.Scan(&cid); err == nil {
					chatIDs = append(chatIDs, cid)
				}
			}
			cRows.Close()
			if chatIDs == nil {
				chatIDs = []string{}
			}
			f.ChatIDs = chatIDs
		} else {
			f.ChatIDs = []string{}
		}

		folders = append(folders, f)
	}

	if folders == nil {
		folders = []models.ChatFolder{}
	}
	return folders, nil
}

type CreateFolderReq struct {
	Name        string   `json:"name"`
	Icon        string   `json:"icon"`
	FilterFlags int      `json:"filterFlags"`
	ChatIDs     []string `json:"chatIds"`
}

func (s *Service) CreateFolder(userID string, req CreateFolderReq) (*models.ChatFolder, error) {
	id := uuid.New().String()
	now := time.Now().UnixMilli()

	var maxOrder sql.NullInt64
	_ = s.db.QueryRow(`SELECT MAX(folder_order) FROM chat_folders WHERE user_id = ?`, userID).Scan(&maxOrder)
	nextOrder := 0
	if maxOrder.Valid {
		nextOrder = int(maxOrder.Int64) + 1
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`INSERT INTO chat_folders (id, user_id, name, icon, filter_flags, folder_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, req.Name, req.Icon, req.FilterFlags, nextOrder, now, now)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, cid := range req.ChatIDs {
		_, _ = tx.Exec(`INSERT OR IGNORE INTO chat_folder_chats (folder_id, chat_id) VALUES (?, ?)`, id, cid)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	chatIDs := req.ChatIDs
	if chatIDs == nil {
		chatIDs = []string{}
	}

	return &models.ChatFolder{
		ID:          id,
		UserID:      userID,
		Name:        req.Name,
		Icon:        req.Icon,
		FilterFlags: req.FilterFlags,
		FolderOrder: nextOrder,
		ChatIDs:     chatIDs,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

type UpdateFolderReq struct {
	Name        *string   `json:"name,omitempty"`
	Icon        *string   `json:"icon,omitempty"`
	FilterFlags *int      `json:"filterFlags,omitempty"`
	FolderOrder *int      `json:"folderOrder,omitempty"`
	ChatIDs     *[]string `json:"chatIds,omitempty"`
}

func (s *Service) UpdateFolder(id, userID string, req UpdateFolderReq) (*models.ChatFolder, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()

	if req.Name != nil {
		_, _ = tx.Exec(`UPDATE chat_folders SET name = ?, updated_at = ? WHERE id = ? AND user_id = ?`, *req.Name, now, id, userID)
	}
	if req.Icon != nil {
		_, _ = tx.Exec(`UPDATE chat_folders SET icon = ?, updated_at = ? WHERE id = ? AND user_id = ?`, *req.Icon, now, id, userID)
	}
	if req.FilterFlags != nil {
		_, _ = tx.Exec(`UPDATE chat_folders SET filter_flags = ?, updated_at = ? WHERE id = ? AND user_id = ?`, *req.FilterFlags, now, id, userID)
	}
	if req.FolderOrder != nil {
		_, _ = tx.Exec(`UPDATE chat_folders SET folder_order = ?, updated_at = ? WHERE id = ? AND user_id = ?`, *req.FolderOrder, now, id, userID)
	}

	if req.ChatIDs != nil {
		_, _ = tx.Exec(`DELETE FROM chat_folder_chats WHERE folder_id = ?`, id)
		for _, cid := range *req.ChatIDs {
			_, _ = tx.Exec(`INSERT OR IGNORE INTO chat_folder_chats (folder_id, chat_id) VALUES (?, ?)`, id, cid)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	folders, err := s.GetFolders(userID)
	if err != nil {
		return nil, err
	}

	for _, f := range folders {
		if f.ID == id {
			return &f, nil
		}
	}

	return nil, sql.ErrNoRows
}

func (s *Service) DeleteFolder(id, userID string) error {
	_, err := s.db.Exec(`DELETE FROM chat_folders WHERE id = ? AND user_id = ?`, id, userID)
	return err
}
