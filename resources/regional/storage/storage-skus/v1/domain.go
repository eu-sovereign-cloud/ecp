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

// StorageSKUDomain represents the domain model for a storage SKU.
type StorageSKUDomain struct {
	domain.RegionalMetadata
	Spec StorageSKUSpecDomain
}

// StorageSKUSpecDomain defines the specification for a storage SKU.
type StorageSKUSpecDomain struct {
	Iops          int64
	MinVolumeSize int64
	Type          string
}
