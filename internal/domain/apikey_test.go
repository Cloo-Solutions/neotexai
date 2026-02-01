package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIKey(t *testing.T) {
	now := time.Now()
	apiKey := NewAPIKey("key1", "org1", "Test Key", "hash123", now, nil)

	assert.Equal(t, "key1", apiKey.ID)
	assert.Equal(t, "org1", apiKey.OrgID)
	assert.Equal(t, "Test Key", apiKey.Name)
	assert.Equal(t, "hash123", apiKey.KeyHash)
	assert.Equal(t, now, apiKey.CreatedAt)
	assert.Nil(t, apiKey.RevokedAt)
}

func TestNewAPIKeyWithRevocation(t *testing.T) {
	now := time.Now()
	revokedAt := now.Add(24 * time.Hour)
	apiKey := NewAPIKey("key1", "org1", "Test Key", "hash123", now, &revokedAt)

	assert.Equal(t, "key1", apiKey.ID)
	assert.NotNil(t, apiKey.RevokedAt)
	assert.Equal(t, revokedAt, *apiKey.RevokedAt)
}

func TestValidateAPIKey(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		apiKey  *APIKey
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid api key",
			apiKey: &APIKey{
				ID:        "key1",
				OrgID:     "org1",
				Name:      "Test Key",
				KeyHash:   "hash123",
				CreatedAt: now,
				RevokedAt: nil,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			apiKey: &APIKey{
				OrgID:     "org1",
				Name:      "Test Key",
				KeyHash:   "hash123",
				CreatedAt: now,
				RevokedAt: nil,
			},
			wantErr: true,
			errMsg:  "ID",
		},
		{
			name: "missing OrgID",
			apiKey: &APIKey{
				ID:        "key1",
				Name:      "Test Key",
				KeyHash:   "hash123",
				CreatedAt: now,
				RevokedAt: nil,
			},
			wantErr: true,
			errMsg:  "OrgID",
		},
		{
			name: "missing Name",
			apiKey: &APIKey{
				ID:        "key1",
				OrgID:     "org1",
				KeyHash:   "hash123",
				CreatedAt: now,
				RevokedAt: nil,
			},
			wantErr: true,
			errMsg:  "Name",
		},
		{
			name: "missing KeyHash",
			apiKey: &APIKey{
				ID:        "key1",
				OrgID:     "org1",
				Name:      "Test Key",
				CreatedAt: now,
				RevokedAt: nil,
			},
			wantErr: true,
			errMsg:  "KeyHash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKey(tt.apiKey)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAPIKeyIsRevoked(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		apiKey   *APIKey
		expected bool
	}{
		{
			name: "not revoked",
			apiKey: &APIKey{
				ID:        "key1",
				OrgID:     "org1",
				Name:      "Test Key",
				KeyHash:   "hash123",
				CreatedAt: now,
				RevokedAt: nil,
			},
			expected: false,
		},
		{
			name: "revoked",
			apiKey: &APIKey{
				ID:        "key1",
				OrgID:     "org1",
				Name:      "Test Key",
				KeyHash:   "hash123",
				CreatedAt: now,
				RevokedAt: &now,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.apiKey.IsRevoked())
		})
	}
}
