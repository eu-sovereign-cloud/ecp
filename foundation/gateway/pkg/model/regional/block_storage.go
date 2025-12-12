package regional

import (
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// BlockStorageDomain represents the domain model for a block storage instance.
type BlockStorageDomain struct {
	model.Metadata
	Spec   BlockStorageSpec
	Status *BlockStorageStatus
}

// BlockStorageSpec defines the specification for a block storage instance.
type BlockStorageSpec struct {
	SizeGB         int
	SkuRef         genv1.Reference
	SourceImageRef *genv1.Reference
}

// BlockStorageStatus defines the status for a block storage instance.
type BlockStorageStatus struct {
	AttachedTo *genv1.Reference
	Conditions []genv1.StatusCondition
	SizeGB     int
	State      *genv1.ResourceState
}
