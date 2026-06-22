// Package domain defines the block storage resource domain model and identity constants.
package v1

import "github.com/eu-sovereign-cloud/ecp/resources/common/domain"

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
	SkuRef         domain.ReferenceDomain
	SourceImageRef *domain.ReferenceDomain
}

// BlockStorageStatus defines the status for a block storage instance.
type BlockStorageStatus struct {
	AttachedTo *domain.ReferenceDomain
	SizeGB     int
	domain.StatusDomain
}
