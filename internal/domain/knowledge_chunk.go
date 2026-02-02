package domain

import "time"

// KnowledgeChunk represents a chunked segment of a knowledge item for search.
type KnowledgeChunk struct {
	ID          string
	KnowledgeID string
	OrgID       string
	ProjectID   string
	Type        KnowledgeType
	Status      KnowledgeStatus
	Title       string
	Summary     string
	Scope       string
	ChunkIndex  int
	Content     string
	Embedding   []float32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
