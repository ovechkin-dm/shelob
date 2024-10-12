package crawler

import (
	"fmt"
	"os"
	"path/filepath"
)

type ContentRepository interface {
	Save(path string, content string) error
	Exists(path string) bool
	GetData(path string) (string, error)
}

type FileSystemRepository struct{}

func (f *FileSystemRepository) Save(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (f *FileSystemRepository) Exists(path string) bool {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false
	}

	return err == nil
}

func (f *FileSystemRepository) GetData(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

func NewFileSystemRepository() *FileSystemRepository {
	return &FileSystemRepository{}
}
