package regional

import (
	"time"
)

// ResourceStateDomain represents the current phase of a resource lifecycle.
type ResourceStateDomain string

const (
	ResourceStatePending   ResourceStateDomain = "pending"
	ResourceStateCreating  ResourceStateDomain = "creating"
	ResourceStateActive    ResourceStateDomain = "active"
	ResourceStateUpdating  ResourceStateDomain = "updating"
	ResourceStateDeleting  ResourceStateDomain = "deleting"
	ResourceStateSuspended ResourceStateDomain = "suspended"
	ResourceStateError     ResourceStateDomain = "error"
)

// StatusDomain represents the common status attributes of a regional resource. Cannot be directly mapped to schema.Status,
// since <Resource>Status does not embed schema.Status. This is purely for reducing code duplication in regional resource domains.
type StatusDomain struct {
	State      *ResourceStateDomain
	Conditions []StatusConditionDomain
}

// StatusConditionDomain describes a single state condition of a regional resource's status at a certain point in time.
type StatusConditionDomain struct {
	// LastTransitionAt is the last time the condition transitioned from one status to another.
	LastTransitionAt time.Time
	// Message is a human-readable message indicating details about the transition.
	Message string
	// Reason for the condition's last transition in CamelCase.
	Reason string
	// State is the current phase of the resource.
	State ResourceStateDomain
	// Type of condition (provider-specific).
	Type string
}
