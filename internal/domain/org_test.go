package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrganization(t *testing.T) {
	now := time.Now()
	org := NewOrganization("org1", "Test Org", now)

	assert.Equal(t, "org1", org.ID)
	assert.Equal(t, "Test Org", org.Name)
	assert.Equal(t, now, org.CreatedAt)
}

func TestValidateOrganization(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		org     *Organization
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid organization",
			org: &Organization{
				ID:        "org1",
				Name:      "Test Org",
				CreatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			org: &Organization{
				Name:      "Test Org",
				CreatedAt: now,
			},
			wantErr: true,
			errMsg:  "ID",
		},
		{
			name: "missing Name",
			org: &Organization{
				ID:        "org1",
				CreatedAt: now,
			},
			wantErr: true,
			errMsg:  "Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrganization(tt.org)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
