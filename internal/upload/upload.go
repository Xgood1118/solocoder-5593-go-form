package upload

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	baseDir string
}

func NewService(baseDir string) (*Service, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &Service{baseDir: baseDir}, nil
}

func (s *Service) UploadFile(formID string, fileName string, file io.Reader) (string, int64, error) {
	now := time.Now()
	dateDir := now.Format("2006-01")

	dir := filepath.Join(s.baseDir, formID, dateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", 0, err
	}

	ext := filepath.Ext(fileName)
	newName := uuid.New().String() + ext
	fullPath := filepath.Join(dir, newName)

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", 0, err
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		return "", 0, err
	}

	relPath := filepath.Join(formID, dateDir, newName)
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	return relPath, size, nil
}

func (s *Service) GetFilePath(relPath string) string {
	return filepath.Join(s.baseDir, relPath)
}

func (s *Service) DeleteFile(relPath string) error {
	fullPath := filepath.Join(s.baseDir, relPath)
	return os.Remove(fullPath)
}

func (s *Service) ListFormFiles(formID string) ([]string, error) {
	var files []string
	formDir := filepath.Join(s.baseDir, formID)

	if _, err := os.Stat(formDir); os.IsNotExist(err) {
		return files, nil
	}

	err := filepath.Walk(formDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(s.baseDir, path)
			files = append(files, strings.ReplaceAll(rel, "\\", "/"))
		}
		return nil
	})

	return files, err
}

func (s *Service) ValidateFileSize(relPath string, maxSize int64) (bool, error) {
	fullPath := filepath.Join(s.baseDir, relPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return false, err
	}
	return info.Size() <= maxSize, nil
}

func GetFileExt(fileName string) string {
	return strings.ToLower(filepath.Ext(fileName))
}

func (s *Service) GetFileSize(relPath string) (int64, error) {
	fullPath := filepath.Join(s.baseDir, relPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (s *Service) GetFileURL(relPath string) string {
	return fmt.Sprintf("/uploads/%s", relPath)
}
