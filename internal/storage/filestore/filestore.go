package filestore

import (
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	rootDir string
}

func New(rootDir string) *Store {
	return &Store{rootDir: rootDir}
}

func (s *Store) FinalPath(kbID, documentID, sha256, originalName string) string {
	return BuildDocumentPath(s.rootDir, kbID, documentID, sha256, originalName)
}

func (s *Store) Move(tempPath, finalPath string) error {
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("create file dir: %w", err)
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return fmt.Errorf("move file: %w", err)
	}
	return nil
}

func (s *Store) Remove(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}
