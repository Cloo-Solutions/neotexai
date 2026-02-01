package domain

import (
	"fmt"
	"time"
)

// Organization represents a tenant in the system
type Organization struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

// NewOrganization creates a new Organization instance
func NewOrganization(id, name string, createdAt time.Time) *Organization {
	return &Organization{
		ID:        id,
		Name:      name,
		CreatedAt: createdAt,
	}
}

// ValidateOrganization validates an Organization instance
func ValidateOrganization(o *Organization) error {
	if o == nil {
		return fmt.Errorf("organization cannot be nil")
	}

	if o.ID == "" {
		return fmt.Errorf("organization ID is required")
	}

	if o.Name == "" {
		return fmt.Errorf("organization Name is required")
	}

	return nil
}
