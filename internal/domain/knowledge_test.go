package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnowledgeTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  KnowledgeType
		expected string
	}{
		{"Guideline", KnowledgeTypeGuideline, "guideline"},
		{"Learning", KnowledgeTypeLearning, "learning"},
		{"Decision", KnowledgeTypeDecision, "decision"},
		{"Template", KnowledgeTypeTemplate, "template"},
		{"Checklist", KnowledgeTypeChecklist, "checklist"},
		{"Snippet", KnowledgeTypeSnippet, "snippet"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.typeVal))
		})
	}
}

func TestKnowledgeStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   KnowledgeStatus
		expected string
	}{
		{"Draft", KnowledgeStatusDraft, "draft"},
		{"Approved", KnowledgeStatusApproved, "approved"},
		{"Deprecated", KnowledgeStatusDeprecated, "deprecated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestNewKnowledge(t *testing.T) {
	now := time.Now()
	knowledge := NewKnowledge(
		"k1",
		"org1",
		"proj1",
		KnowledgeTypeGuideline,
		KnowledgeStatusDraft,
		"Test Title",
		"Test Summary",
		"# Test Body",
		now,
		now,
	)

	assert.Equal(t, "k1", knowledge.ID)
	assert.Equal(t, "org1", knowledge.OrgID)
	assert.Equal(t, "proj1", knowledge.ProjectID)
	assert.Equal(t, KnowledgeTypeGuideline, knowledge.Type)
	assert.Equal(t, KnowledgeStatusDraft, knowledge.Status)
	assert.Equal(t, "Test Title", knowledge.Title)
	assert.Equal(t, "Test Summary", knowledge.Summary)
	assert.Equal(t, "# Test Body", knowledge.BodyMD)
	assert.Equal(t, now, knowledge.CreatedAt)
	assert.Equal(t, now, knowledge.UpdatedAt)
	assert.Equal(t, "", knowledge.Scope)
}

func TestValidateKnowledge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		knowledge *Knowledge
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid knowledge",
			knowledge: &Knowledge{
				ID:        "k1",
				OrgID:     "org1",
				ProjectID: "proj1",
				Type:      KnowledgeTypeGuideline,
				Status:    KnowledgeStatusDraft,
				Title:     "Test Title",
				Summary:   "Test Summary",
				BodyMD:    "# Test Body",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			knowledge: &Knowledge{
				OrgID:     "org1",
				ProjectID: "proj1",
				Type:      KnowledgeTypeGuideline,
				Status:    KnowledgeStatusDraft,
				Title:     "Test Title",
				Summary:   "Test Summary",
				BodyMD:    "# Test Body",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "ID",
		},
		{
			name: "missing OrgID",
			knowledge: &Knowledge{
				ID:        "k1",
				ProjectID: "proj1",
				Type:      KnowledgeTypeGuideline,
				Status:    KnowledgeStatusDraft,
				Title:     "Test Title",
				Summary:   "Test Summary",
				BodyMD:    "# Test Body",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "OrgID",
		},
		{
			name: "missing Title",
			knowledge: &Knowledge{
				ID:        "k1",
				OrgID:     "org1",
				ProjectID: "proj1",
				Type:      KnowledgeTypeGuideline,
				Status:    KnowledgeStatusDraft,
				Summary:   "Test Summary",
				BodyMD:    "# Test Body",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "Title",
		},
		{
			name: "missing BodyMD",
			knowledge: &Knowledge{
				ID:        "k1",
				OrgID:     "org1",
				ProjectID: "proj1",
				Type:      KnowledgeTypeGuideline,
				Status:    KnowledgeStatusDraft,
				Title:     "Test Title",
				Summary:   "Test Summary",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "BodyMD",
		},
		{
			name: "invalid Type",
			knowledge: &Knowledge{
				ID:        "k1",
				OrgID:     "org1",
				ProjectID: "proj1",
				Type:      KnowledgeType("invalid"),
				Status:    KnowledgeStatusDraft,
				Title:     "Test Title",
				Summary:   "Test Summary",
				BodyMD:    "# Test Body",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "Type",
		},
		{
			name: "invalid Status",
			knowledge: &Knowledge{
				ID:        "k1",
				OrgID:     "org1",
				ProjectID: "proj1",
				Type:      KnowledgeTypeGuideline,
				Status:    KnowledgeStatus("invalid"),
				Title:     "Test Title",
				Summary:   "Test Summary",
				BodyMD:    "# Test Body",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "Status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKnowledge(tt.knowledge)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestKnowledgeVersionCreation(t *testing.T) {
	now := time.Now()

	version := NewKnowledgeVersion(
		"v1",
		"k1",
		1,
		"Test Title",
		"Test Summary",
		"# Test Body",
		now,
	)

	assert.Equal(t, "v1", version.ID)
	assert.Equal(t, "k1", version.KnowledgeID)
	assert.Equal(t, int64(1), version.VersionNumber)
	assert.Equal(t, "Test Title", version.Title)
	assert.Equal(t, "Test Summary", version.Summary)
	assert.Equal(t, "# Test Body", version.BodyMD)
	assert.Equal(t, now, version.CreatedAt)
}

func TestValidateKnowledgeVersion(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		version *KnowledgeVersion
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid version",
			version: &KnowledgeVersion{
				ID:            "v1",
				KnowledgeID:   "k1",
				VersionNumber: 1,
				Title:         "Test Title",
				Summary:       "Test Summary",
				BodyMD:        "# Test Body",
				CreatedAt:     now,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			version: &KnowledgeVersion{
				KnowledgeID:   "k1",
				VersionNumber: 1,
				Title:         "Test Title",
				Summary:       "Test Summary",
				BodyMD:        "# Test Body",
				CreatedAt:     now,
			},
			wantErr: true,
			errMsg:  "ID",
		},
		{
			name: "missing KnowledgeID",
			version: &KnowledgeVersion{
				ID:            "v1",
				VersionNumber: 1,
				Title:         "Test Title",
				Summary:       "Test Summary",
				BodyMD:        "# Test Body",
				CreatedAt:     now,
			},
			wantErr: true,
			errMsg:  "KnowledgeID",
		},
		{
			name: "invalid VersionNumber",
			version: &KnowledgeVersion{
				ID:            "v1",
				KnowledgeID:   "k1",
				VersionNumber: 0,
				Title:         "Test Title",
				Summary:       "Test Summary",
				BodyMD:        "# Test Body",
				CreatedAt:     now,
			},
			wantErr: true,
			errMsg:  "VersionNumber",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKnowledgeVersion(tt.version)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
