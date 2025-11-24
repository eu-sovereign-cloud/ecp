package regional

import "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"

// StorageSKUDomain represents the domain model for a storage SKU.
type StorageSKUDomain struct {
	model.Metadata
	Spec StorageSKUSpec
}

// StorageSKUSpec defines the specification for a storage SKU.
type StorageSKUSpec struct {
	Iops          int64
	MinVolumeSize int64
	Type          string
}
