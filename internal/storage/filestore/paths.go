package filestore

import (
	"path/filepath"
	"strings"
)

func BuildDocumentPath(rootDir, kbID, documentID, sha256, originalName string) string {
	safeName := sanitizeFilename(originalName)
	return filepath.Join(rootDir, kbID, documentID, sha256+"_"+safeName)
}

func sanitizeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "" || name == "." {
		return "upload.bin"
	}
	return name
}
