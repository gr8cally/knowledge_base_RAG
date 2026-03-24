package ingest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LoadResult struct {
	Text       string
	ParserUsed string
}

func LoadDocument(ctx context.Context, path, mimeType string, ocrEnabled bool, ocrLang string) (LoadResult, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".markdown":
		data, err := os.ReadFile(path)
		if err != nil {
			return LoadResult{}, fmt.Errorf("read text file: %w", err)
		}
		parser := "text"
		if ext == ".md" || ext == ".markdown" {
			parser = "markdown"
		}
		return LoadResult{Text: string(data), ParserUsed: parser}, nil
	case ".png", ".jpg", ".jpeg", ".tif", ".tiff":
		if !ocrEnabled {
			return LoadResult{}, fmt.Errorf("ocr disabled for image upload")
		}
		text, err := RunOCR(ctx, path, ocrLang)
		if err != nil {
			return LoadResult{}, err
		}
		return LoadResult{Text: text, ParserUsed: "ocr"}, nil
	default:
		if strings.HasPrefix(mimeType, "text/") {
			data, err := os.ReadFile(path)
			if err != nil {
				return LoadResult{}, fmt.Errorf("read text file: %w", err)
			}
			return LoadResult{Text: string(data), ParserUsed: "text"}, nil
		}
		return LoadResult{}, fmt.Errorf("unsupported file type: %s", ext)
	}
}
