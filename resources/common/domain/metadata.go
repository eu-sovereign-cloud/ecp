package domain

import (
	"time"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
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

// RegionalMetadata carries common resource identity and classification data used in regional domain models.
// It embeds CommonMetadata and resource.Scope to provide tenant and workspace access.
type RegionalMetadata struct {
	CommonMetadata
	resource.Scope

	Labels      map[string]string
	Annotations map[string]string
	Extensions  map[string]string
	Region      string
}
