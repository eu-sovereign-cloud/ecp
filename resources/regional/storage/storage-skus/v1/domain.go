// Package domain defines the storage SKU resource domain model and identity constants.
package v1

import "github.com/eu-sovereign-cloud/ecp/resources/common/domain"

// Identity constants for the storage SKU resource.
const (
	Kind       = "StorageSKU"
	Resource   = "storage-skus"
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
	Iops          int64
	MinVolumeSize int64
	Type          string
}
