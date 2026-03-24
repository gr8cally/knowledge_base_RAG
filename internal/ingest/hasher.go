package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
)

func CopyMultipartToTempAndHash(src multipart.File) (string, string, int64, error) {
	tmp, err := os.CreateTemp("", "kb-upload-*")
	if err != nil {
		return "", "", 0, fmt.Errorf("create temp file: %w", err)
	}
	defer tmp.Close()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hasher), src)
	if err != nil {
		_ = os.Remove(tmp.Name())
		return "", "", 0, fmt.Errorf("copy upload: %w", err)
	}

	return tmp.Name(), hex.EncodeToString(hasher.Sum(nil)), written, nil
}
