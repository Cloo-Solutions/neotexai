package domain

import (
	"fmt"
	"time"
)

// Project represents a project scoped to an organization
type Project struct {
	ID        string
	OrgID     string
	Name      string
	CreatedAt time.Time
}

// NewProject creates a new Project instance
func NewProject(id, orgID, name string, createdAt time.Time) *Project {
	return &Project{
		ID:        id,
		OrgID:     orgID,
		Name:      name,
		CreatedAt: createdAt,
	}
}

// ValidateProject validates a Project instance
func ValidateProject(p *Project) error {
	if p == nil {
		return fmt.Errorf("project cannot be nil")
	}

	if p.ID == "" {
		return fmt.Errorf("project ID is required")
	}

	if p.OrgID == "" {
		return fmt.Errorf("project OrgID is required")
	}

	if p.Name == "" {
		return fmt.Errorf("project Name is required")
	}

	return nil
}
