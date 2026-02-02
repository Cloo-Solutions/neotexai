package service

import (
	"strings"
	"unicode"
)

// ChunkConfig controls chunking for knowledge embeddings.
type ChunkConfig struct {
	MaxChars  int
	MinChars  int
	Overlap   int
	MaxChunks int
}

// DefaultChunkConfig provides sane defaults for chunking.
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		MaxChars:  1200,
		MinChars:  400,
		Overlap:   200,
		MaxChunks: 40,
	}
}

func chunkText(text string, cfg ChunkConfig) []string {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return nil
	}
	if cfg.MaxChars <= 0 {
		cfg = DefaultChunkConfig()
	}
	runes := []rune(clean)
	if len(runes) <= cfg.MaxChars {
		return []string{clean}
	}

	chunks := make([]string, 0, 8)
	start := 0
	for start < len(runes) {
		if cfg.MaxChunks > 0 && len(chunks) >= cfg.MaxChunks {
			break
		}

		end := start + cfg.MaxChars
		if end > len(runes) {
			end = len(runes)
		}

		if end < len(runes) {
			cut := end
			minCut := start + cfg.MinChars
			if minCut > end {
				minCut = start
			}
			for i := end; i > minCut; i-- {
				if unicode.IsSpace(runes[i-1]) {
					cut = i
					break
				}
			}
			end = cut
		}

		if end <= start {
			break
		}

		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		if end >= len(runes) {
			break
		}

		nextStart := end
		if cfg.Overlap > 0 {
			if end-start > cfg.Overlap {
				nextStart = end - cfg.Overlap
			}
		}
		if nextStart <= start {
			nextStart = end
		}
		start = nextStart
	}

	return chunks
}
