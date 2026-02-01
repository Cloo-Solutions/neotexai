package domain

import (
	"fmt"
	"time"
)

// Asset represents a binary or file-based reference associated with knowledge
type Asset struct {
	ID          string
	OrgID       string
	ProjectID   string
	Filename    string
	MimeType    string
	SHA256      string
	StorageKey  string
	Keywords    []string
	Description string
	Embedding   []float32
	CreatedAt   time.Time
}

// NewAsset creates a new Asset instance
func NewAsset(
	id, orgID, projectID string,
	filename, mimeType, sha256, storageKey string,
	keywords []string,
	description string,
	createdAt time.Time,
) *Asset {
	return &Asset{
		ID:          id,
		OrgID:       orgID,
		ProjectID:   projectID,
		Filename:    filename,
		MimeType:    mimeType,
		SHA256:      sha256,
		StorageKey:  storageKey,
		Keywords:    keywords,
		Description: description,
		CreatedAt:   createdAt,
	}
}

// ValidateAsset validates an Asset instance
func ValidateAsset(a *Asset) error {
	if a == nil {
		return fmt.Errorf("asset cannot be nil")
	}

	if a.ID == "" {
		return fmt.Errorf("asset ID is required")
	}

	if a.OrgID == "" {
		return fmt.Errorf("asset OrgID is required")
	}

	if a.Filename == "" {
		return fmt.Errorf("asset Filename is required")
	}

	if a.MimeType == "" {
		return fmt.Errorf("asset MimeType is required")
	}

	if a.SHA256 == "" {
		return fmt.Errorf("asset SHA256 is required")
	}

	if a.StorageKey == "" {
		return fmt.Errorf("asset StorageKey is required")
	}

	return nil
}
