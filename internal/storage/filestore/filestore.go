package filestore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
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

func (s *Store) RootDir() string {
	return s.rootDir
}

func (s *Store) Move(tempPath, finalPath string) error {
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("create file dir: %w", err)
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		if !isCrossDeviceRename(err) {
			return fmt.Errorf("move file: %w", err)
		}
		if err := copyAndRemove(tempPath, finalPath); err != nil {
			return fmt.Errorf("copy file across devices: %w", err)
		}
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

func isCrossDeviceRename(err error) bool {
	linkErr, ok := err.(*os.LinkError)
	if !ok {
		return false
	}
	return linkErr.Err == syscall.EXDEV
}

func copyAndRemove(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}

	return os.Remove(srcPath)
}
