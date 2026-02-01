package domain

import (
	"fmt"
	"time"
)

// APIKey represents an API key for authentication
type APIKey struct {
	ID        string
	OrgID     string
	Name      string
	KeyHash   string // Never store plaintext keys
	CreatedAt time.Time
	RevokedAt *time.Time
}

// NewAPIKey creates a new APIKey instance
func NewAPIKey(id, orgID, name, keyHash string, createdAt time.Time, revokedAt *time.Time) *APIKey {
	return &APIKey{
		ID:        id,
		OrgID:     orgID,
		Name:      name,
		KeyHash:   keyHash,
		CreatedAt: createdAt,
		RevokedAt: revokedAt,
	}
}

// IsRevoked returns true if the API key has been revoked
func (a *APIKey) IsRevoked() bool {
	return a.RevokedAt != nil
}

// ValidateAPIKey validates an APIKey instance
func ValidateAPIKey(a *APIKey) error {
	if a == nil {
		return fmt.Errorf("api key cannot be nil")
	}

	if a.ID == "" {
		return fmt.Errorf("api key ID is required")
	}

	if a.OrgID == "" {
		return fmt.Errorf("api key OrgID is required")
	}

	if a.Name == "" {
		return fmt.Errorf("api key Name is required")
	}

	if a.KeyHash == "" {
		return fmt.Errorf("api key KeyHash is required")
	}

	return nil
}
