package regional

import "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"

/*
 *
 * NOTE: This model is not complete nor ready for production usage.
 *       It's only for exploratory development.
 *
 */

type StorageBlockStorageDomain struct {
	model.Metadata
	Spec   StorageBlockStorageSpec
	Status StorageBlockStorageStatus
}

type StorageBlockStorageSpec struct {
	// SizeGB Size of the block storage in GB.
	SizeGB int

	// SkuType to the SKU of the block storage.
	SkuType string

	// SourceImage ID to the source image used as the base for creating the block storage.
	SourceImageID *string
}

type StorageBlockStorageStatus struct {
	model.ResourceStatus
	StorageBlockStorageSpec
}
