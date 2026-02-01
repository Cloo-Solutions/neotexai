package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAsset(t *testing.T) {
	now := time.Now()
	asset := NewAsset(
		"a1",
		"org1",
		"proj1",
		"document.pdf",
		"application/pdf",
		"abc123def456",
		"s3://bucket/document.pdf",
		[]string{"pdf", "document"},
		"A test document",
		now,
	)

	assert.Equal(t, "a1", asset.ID)
	assert.Equal(t, "org1", asset.OrgID)
	assert.Equal(t, "proj1", asset.ProjectID)
	assert.Equal(t, "document.pdf", asset.Filename)
	assert.Equal(t, "application/pdf", asset.MimeType)
	assert.Equal(t, "abc123def456", asset.SHA256)
	assert.Equal(t, "s3://bucket/document.pdf", asset.StorageKey)
	assert.Equal(t, []string{"pdf", "document"}, asset.Keywords)
	assert.Equal(t, "A test document", asset.Description)
	assert.Equal(t, now, asset.CreatedAt)
}

func TestValidateAsset(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		asset   *Asset
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid asset",
			asset: &Asset{
				ID:          "a1",
				OrgID:       "org1",
				ProjectID:   "proj1",
				Filename:    "document.pdf",
				MimeType:    "application/pdf",
				SHA256:      "abc123def456",
				StorageKey:  "s3://bucket/document.pdf",
				Keywords:    []string{"pdf"},
				Description: "A test document",
				CreatedAt:   now,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			asset: &Asset{
				OrgID:       "org1",
				ProjectID:   "proj1",
				Filename:    "document.pdf",
				MimeType:    "application/pdf",
				SHA256:      "abc123def456",
				StorageKey:  "s3://bucket/document.pdf",
				Keywords:    []string{"pdf"},
				Description: "A test document",
				CreatedAt:   now,
			},
			wantErr: true,
			errMsg:  "ID",
		},
		{
			name: "missing OrgID",
			asset: &Asset{
				ID:          "a1",
				ProjectID:   "proj1",
				Filename:    "document.pdf",
				MimeType:    "application/pdf",
				SHA256:      "abc123def456",
				StorageKey:  "s3://bucket/document.pdf",
				Keywords:    []string{"pdf"},
				Description: "A test document",
				CreatedAt:   now,
			},
			wantErr: true,
			errMsg:  "OrgID",
		},
		{
			name: "missing Filename",
			asset: &Asset{
				ID:          "a1",
				OrgID:       "org1",
				ProjectID:   "proj1",
				MimeType:    "application/pdf",
				SHA256:      "abc123def456",
				StorageKey:  "s3://bucket/document.pdf",
				Keywords:    []string{"pdf"},
				Description: "A test document",
				CreatedAt:   now,
			},
			wantErr: true,
			errMsg:  "Filename",
		},
		{
			name: "missing MimeType",
			asset: &Asset{
				ID:          "a1",
				OrgID:       "org1",
				ProjectID:   "proj1",
				Filename:    "document.pdf",
				SHA256:      "abc123def456",
				StorageKey:  "s3://bucket/document.pdf",
				Keywords:    []string{"pdf"},
				Description: "A test document",
				CreatedAt:   now,
			},
			wantErr: true,
			errMsg:  "MimeType",
		},
		{
			name: "missing SHA256",
			asset: &Asset{
				ID:          "a1",
				OrgID:       "org1",
				ProjectID:   "proj1",
				Filename:    "document.pdf",
				MimeType:    "application/pdf",
				StorageKey:  "s3://bucket/document.pdf",
				Keywords:    []string{"pdf"},
				Description: "A test document",
				CreatedAt:   now,
			},
			wantErr: true,
			errMsg:  "SHA256",
		},
		{
			name: "missing StorageKey",
			asset: &Asset{
				ID:          "a1",
				OrgID:       "org1",
				ProjectID:   "proj1",
				Filename:    "document.pdf",
				MimeType:    "application/pdf",
				SHA256:      "abc123def456",
				Keywords:    []string{"pdf"},
				Description: "A test document",
				CreatedAt:   now,
			},
			wantErr: true,
			errMsg:  "StorageKey",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAsset(tt.asset)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
