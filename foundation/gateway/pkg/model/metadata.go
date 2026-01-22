package model

import (
	"time"
)

// CommonMetadata carries common resource identity and classification data used in domain models.
type CommonMetadata struct {
	Name            string
	Provider        string
	ResourceVersion string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

func (m *CommonMetadata) GetName() string    { return m.Name }
func (m *CommonMetadata) GetVersion() string { return m.ResourceVersion }

// Metadata carries common resource identity and classification data used in global domain models.
// It is an alias for CommonMetadata to simplify references and allow providing sentinel methods for tenant and workspace.
type Metadata struct{ CommonMetadata }

func (m *Metadata) GetTenant() string    { return "" }
func (m *Metadata) GetWorkspace() string { return "" }
