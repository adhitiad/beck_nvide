package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// Storage defines interface for storing files
type Storage interface {
	Upload(ctx context.Context, key string, body io.Reader) (string, error)
	Delete(ctx context.Context, key string) error
	GetURL(key string) string
}

// LocalStorage implements Storage using local filesystem
type LocalStorage struct {
	baseDir string
	baseURL string
	logger  *zap.Logger
}

func NewLocalStorage(baseDir, baseURL string, logger *zap.Logger) *LocalStorage {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		logger.Fatal("Failed to create storage directory", zap.Error(err))
	}
	return &LocalStorage{
		baseDir: baseDir,
		baseURL: baseURL,
		logger:  logger,
	}
}

func (s *LocalStorage) Upload(ctx context.Context, key string, body io.Reader) (string, error) {
	fullPath := filepath.Join(s.baseDir, key)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, body); err != nil {
		return "", err
	}

	return s.GetURL(key), nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.baseDir, key)
	// Can also be a directory (like for HLS segments)
	err := os.RemoveAll(fullPath)
	if err != nil {
		s.logger.Error("Failed to delete file/dir", zap.String("path", fullPath), zap.Error(err))
	}
	return err
}

func (s *LocalStorage) GetURL(key string) string {
	// For HLS, this could be the m3u8 file
	return s.baseURL + "/" + key
}
