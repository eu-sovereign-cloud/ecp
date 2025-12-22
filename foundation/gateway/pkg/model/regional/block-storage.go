package regional

import (
	"time"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// ResourceState represents the current phase of a resource lifecycle.
type ResourceState string

const (
	ResourceStatePending   ResourceState = "pending"
	ResourceStateCreating  ResourceState = "creating"
	ResourceStateActive    ResourceState = "active"
	ResourceStateUpdating  ResourceState = "updating"
	ResourceStateDeleting  ResourceState = "deleting"
	ResourceStateSuspended ResourceState = "suspended"
	ResourceStateError     ResourceState = "error"
)

// StatusCondition describes the state of a resource at a certain point.
type StatusCondition struct {
	// LastTransitionAt is the last time the condition transitioned from one status to another.
	LastTransitionAt time.Time
	// Message is a human-readable message indicating details about the transition.
	Message string
	// Reason for the condition's last transition in CamelCase.
	Reason string
	// State is the current phase of the resource.
	State ResourceState
	// Type of condition (provider-specific).
	Type string
}

// BlockStorageDomain represents the domain model for a block storage instance.
type BlockStorageDomain struct {
	model.Metadata
	Spec   BlockStorageSpec
	Status *BlockStorageStatus
}

// BlockStorageSpec defines the specification for a block storage instance.
type BlockStorageSpec struct {
	SizeGB         int
	SkuRef         ReferenceObject
	SourceImageRef *ReferenceObject
}

// BlockStorageStatus defines the status for a block storage instance.
type BlockStorageStatus struct {
	AttachedTo *ReferenceObject
	Conditions []StatusCondition
	SizeGB     int
	State      *ResourceState
}
