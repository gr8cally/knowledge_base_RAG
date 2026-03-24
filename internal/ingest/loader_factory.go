package ingest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

type LoadResult struct {
	Text       string
	ParserUsed string
}

func LoadDocument(ctx context.Context, path, mimeType string, ocrEnabled bool, ocrLang string) (LoadResult, error) {
	ext := strings.ToLower(filepath.Ext(path))
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(800),
		textsplitter.WithChunkOverlap(120),
	)

	switch ext {
	case ".txt", ".md", ".markdown":
		f, err := os.Open(path)
		if err != nil {
			return LoadResult{}, fmt.Errorf("open text file: %w", err)
		}
		defer f.Close()

		docs, err := documentloaders.NewText(f).LoadAndSplit(ctx, splitter)
		if err != nil {
			return LoadResult{}, fmt.Errorf("langchaingo text loader failed: %w", err)
		}

		parser := "text"
		if ext == ".md" || ext == ".markdown" {
			parser = "markdown"
		}
		return LoadResult{Text: joinDocuments(docs), ParserUsed: parser}, nil
	case ".pdf":
		f, err := os.Open(path)
		if err != nil {
			return LoadResult{}, fmt.Errorf("open pdf file: %w", err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return LoadResult{}, fmt.Errorf("stat pdf file: %w", err)
		}

		docs, err := documentloaders.NewPDF(f, info.Size()).LoadAndSplit(ctx, splitter)
		if err != nil {
			return LoadResult{}, fmt.Errorf("langchaingo pdf loader failed: %w", err)
		}
		return LoadResult{Text: joinDocuments(docs), ParserUsed: "pdf"}, nil
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
			f, err := os.Open(path)
			if err != nil {
				return LoadResult{}, fmt.Errorf("open text file: %w", err)
			}
			defer f.Close()

			docs, err := documentloaders.NewText(f).LoadAndSplit(ctx, splitter)
			if err != nil {
				return LoadResult{}, fmt.Errorf("langchaingo text loader failed: %w", err)
			}
			return LoadResult{Text: joinDocuments(docs), ParserUsed: "text"}, nil
		}
		return LoadResult{}, fmt.Errorf("unsupported file type: %s", ext)
	}
}

func joinDocuments(docs []schema.Document) string {
	if len(docs) == 0 {
		return ""
	}

	parts := make([]string, 0, len(docs))
	for _, doc := range docs {
		if doc.PageContent == "" {
			continue
		}
		parts = append(parts, doc.PageContent)
	}
	return strings.Join(parts, "\n")
}
