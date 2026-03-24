package ingest

import "github.com/tmc/langchaingo/textsplitter"

func ChunkText(text string, chunkSize, overlap int) ([]string, error) {
	if text == "" {
		return []string{}, nil
	}

	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(chunkSize),
		textsplitter.WithChunkOverlap(overlap),
	)
	return splitter.SplitText(text)
}
