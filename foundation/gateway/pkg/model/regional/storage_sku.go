package regional

import "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"

// StorageSKUDomain represents the domain model for a storage SKU.
type StorageSKUDomain struct {
	Meta model.Metadata
	Spec StorageSKUSpec
}

// StorageSKUSpec defines the specification for a storage SKU.
type StorageSKUSpec struct {
	Iops          int64
	MinVolumeSize int64
	Type          string
}

// GetName returns the name of the storage SKU domain.
func (s *StorageSKUDomain) GetName() string {
	return s.Meta.Name
}

// GetNamespace returns the namespace of the storage SKU domain.
func (s *StorageSKUDomain) GetNamespace() string {
	return s.Meta.Namespace
}

// SetName sets the name of the storage SKU domain.
func (s *StorageSKUDomain) SetName(name string) {
	s.Meta.Name = name
}

// SetNamespace sets the namespace of the storage SKU domain.
func (s *StorageSKUDomain) SetNamespace(namespace string) {
	// SKUs are cluster scoped; store for completeness
	s.Meta.Namespace = namespace
}
