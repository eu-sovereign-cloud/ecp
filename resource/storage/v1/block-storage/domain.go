// Package blockstorage defines the block storage resource domain model and identity constants.
package blockstorage

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the block storage resource.
const (
	Kind       = "BlockStorage"
	Resource   = "block-storages"
	Group      = "storage.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.storage/v1"
)

// BlockStorage represents the domain model for a block storage instance.
type BlockStorage struct {
	domain.RegionalMetadata
	Spec   BlockStorageSpec
	Status *BlockStorageStatus
}

// BlockStorageSpec defines the specification for a block storage instance.
type BlockStorageSpec struct {
	SizeGB         int
	SkuRef         domain.Reference
	SourceImageRef *domain.Reference
}

// BlockStorageStatus defines the status for a block storage instance.
type BlockStorageStatus struct {
	AttachedTo *domain.Reference
	SizeGB     int
	domain.Status
}
