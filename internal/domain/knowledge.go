package domain

import (
	"fmt"
	"time"
)

// KnowledgeType represents the type of knowledge item
type KnowledgeType string

const (
	KnowledgeTypeGuideline KnowledgeType = "guideline"
	KnowledgeTypeLearning  KnowledgeType = "learning"
	KnowledgeTypeDecision  KnowledgeType = "decision"
	KnowledgeTypeTemplate  KnowledgeType = "template"
	KnowledgeTypeChecklist KnowledgeType = "checklist"
	KnowledgeTypeSnippet   KnowledgeType = "snippet"
)

// KnowledgeStatus represents the status of a knowledge item
type KnowledgeStatus string

const (
	KnowledgeStatusDraft      KnowledgeStatus = "draft"
	KnowledgeStatusApproved   KnowledgeStatus = "approved"
	KnowledgeStatusDeprecated KnowledgeStatus = "deprecated"
)

// Knowledge represents a knowledge item in the system
type Knowledge struct {
	ID        string
	OrgID     string
	ProjectID string
	Type      KnowledgeType
	Status    KnowledgeStatus
	Title     string
	Summary   string
	BodyMD    string
	CreatedAt time.Time
	UpdatedAt time.Time
	Scope     string // Optional scope (file or path)
}

// KnowledgeVersion represents an immutable version of a knowledge item
type KnowledgeVersion struct {
	ID            string
	KnowledgeID   string
	VersionNumber int64
	Title         string
	Summary       string
	BodyMD        string
	CreatedAt     time.Time
}

// NewKnowledge creates a new Knowledge instance
func NewKnowledge(
	id, orgID, projectID string,
	knowledgeType KnowledgeType,
	status KnowledgeStatus,
	title, summary, bodyMD string,
	createdAt, updatedAt time.Time,
) *Knowledge {
	return &Knowledge{
		ID:        id,
		OrgID:     orgID,
		ProjectID: projectID,
		Type:      knowledgeType,
		Status:    status,
		Title:     title,
		Summary:   summary,
		BodyMD:    bodyMD,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Scope:     "",
	}
}

// NewKnowledgeVersion creates a new KnowledgeVersion instance
func NewKnowledgeVersion(
	id, knowledgeID string,
	versionNumber int64,
	title, summary, bodyMD string,
	createdAt time.Time,
) *KnowledgeVersion {
	return &KnowledgeVersion{
		ID:            id,
		KnowledgeID:   knowledgeID,
		VersionNumber: versionNumber,
		Title:         title,
		Summary:       summary,
		BodyMD:        bodyMD,
		CreatedAt:     createdAt,
	}
}

// ValidateKnowledge validates a Knowledge instance
func ValidateKnowledge(k *Knowledge) error {
	if k == nil {
		return fmt.Errorf("knowledge cannot be nil")
	}

	if k.ID == "" {
		return fmt.Errorf("knowledge ID is required")
	}

	if k.OrgID == "" {
		return fmt.Errorf("knowledge OrgID is required")
	}

	if k.Title == "" {
		return fmt.Errorf("knowledge Title is required")
	}

	if k.BodyMD == "" {
		return fmt.Errorf("knowledge BodyMD is required")
	}

	if !isValidKnowledgeType(k.Type) {
		return fmt.Errorf("knowledge Type is invalid: %s", k.Type)
	}

	if !isValidKnowledgeStatus(k.Status) {
		return fmt.Errorf("knowledge Status is invalid: %s", k.Status)
	}

	return nil
}

// ValidateKnowledgeVersion validates a KnowledgeVersion instance
func ValidateKnowledgeVersion(kv *KnowledgeVersion) error {
	if kv == nil {
		return fmt.Errorf("knowledge version cannot be nil")
	}

	if kv.ID == "" {
		return fmt.Errorf("knowledge version ID is required")
	}

	if kv.KnowledgeID == "" {
		return fmt.Errorf("knowledge version KnowledgeID is required")
	}

	if kv.VersionNumber <= 0 {
		return fmt.Errorf("knowledge version VersionNumber must be greater than 0")
	}

	return nil
}

// isValidKnowledgeType checks if a KnowledgeType is valid
func isValidKnowledgeType(t KnowledgeType) bool {
	switch t {
	case KnowledgeTypeGuideline, KnowledgeTypeLearning, KnowledgeTypeDecision,
		KnowledgeTypeTemplate, KnowledgeTypeChecklist, KnowledgeTypeSnippet:
		return true
	}
	return false
}

// isValidKnowledgeStatus checks if a KnowledgeStatus is valid
func isValidKnowledgeStatus(s KnowledgeStatus) bool {
	switch s {
	case KnowledgeStatusDraft, KnowledgeStatusApproved, KnowledgeStatusDeprecated:
		return true
	}
	return false
}
