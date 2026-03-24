package ingest

import (
	"context"
	"fmt"
	"os/exec"
)

func RunOCR(ctx context.Context, path, lang string) (string, error) {
	cmd := exec.CommandContext(ctx, "tesseract", path, "stdout", "-l", lang)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tesseract failed: %w", err)
	}
	return string(out), nil
}
