package ingest

func ChunkText(text string, chunkSize, overlap int) []string {
	if chunkSize <= 0 {
		return nil
	}
	if overlap >= chunkSize {
		overlap = 0
	}
	if text == "" {
		return []string{}
	}

	var chunks []string
	runes := []rune(text)
	step := chunkSize - overlap
	for start := 0; start < len(runes); start += step {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) {
			break
		}
	}
	return chunks
}
