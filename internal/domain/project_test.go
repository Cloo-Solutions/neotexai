package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProject(t *testing.T) {
	now := time.Now()
	project := NewProject("proj1", "org1", "Test Project", now)

	assert.Equal(t, "proj1", project.ID)
	assert.Equal(t, "org1", project.OrgID)
	assert.Equal(t, "Test Project", project.Name)
	assert.Equal(t, now, project.CreatedAt)
}

func TestValidateProject(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		project *Project
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid project",
			project: &Project{
				ID:        "proj1",
				OrgID:     "org1",
				Name:      "Test Project",
				CreatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			project: &Project{
				OrgID:     "org1",
				Name:      "Test Project",
				CreatedAt: now,
			},
			wantErr: true,
			errMsg:  "ID",
		},
		{
			name: "missing OrgID",
			project: &Project{
				ID:        "proj1",
				Name:      "Test Project",
				CreatedAt: now,
			},
			wantErr: true,
			errMsg:  "OrgID",
		},
		{
			name: "missing Name",
			project: &Project{
				ID:        "proj1",
				OrgID:     "org1",
				CreatedAt: now,
			},
			wantErr: true,
			errMsg:  "Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProject(tt.project)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
