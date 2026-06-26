// Package storagesku defines the storage SKU resource domain model and identity constants.
package storagesku

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the storage SKU resource.
const (
	Kind       = "StorageSKU"
	Resource   = "skus"
	Group      = "storage.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.storage/v1"
)

// StorageSKU represents the domain model for a storage SKU.
type StorageSKU struct {
	domain.RegionalMetadata
	Spec StorageSKUSpec
}

// StorageSKUSpec defines the specification for a storage SKU.
type StorageSKUSpec struct {
	IOPS          int64
	MinVolumeSize int64
	Type          string
}
