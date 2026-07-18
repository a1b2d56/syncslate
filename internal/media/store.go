package media

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Store interface {
	Save(fileName string, r io.Reader) (string, error)
	Get(filePath string) (io.ReadCloser, error)
}

type FileSystemStore struct {
	baseDir string
}

func NewFileSystemStore(baseDir string) (*FileSystemStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload dir: %w", err)
	}
	return &FileSystemStore{baseDir: baseDir}, nil
}

func (s *FileSystemStore) Save(fileName string, r io.Reader) (string, error) {
	now := time.Now()
	yearStr := now.Format("2006")
	monthStr := now.Format("01")

	relDir := filepath.Join(yearStr, monthStr)
	targetDir := filepath.Join(s.baseDir, relDir)

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	relPath := filepath.Join(relDir, fileName)
	fullPath := filepath.Join(s.baseDir, relPath)

	out, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, r); err != nil {
		return "", err
	}

	return relPath, nil
}

func (s *FileSystemStore) Get(filePath string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.baseDir, filePath)
	return os.Open(fullPath)
}
