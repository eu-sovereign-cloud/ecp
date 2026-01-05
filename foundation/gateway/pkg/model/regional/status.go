package regional

import (
	"time"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
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
	Message *string
	// Reason for the condition's last transition in CamelCase.
	Reason *string
	// State is the current phase of the resource.package regional
	State ResourceStateDomain
	// Type of condition (provider-specific).
	Type *string
}

// mapResourceStateDomainToAPI maps ResourceStateDomain to a schema.ResourceState.
func mapResourceStateDomainToAPI(domain ResourceStateDomain) schema.ResourceState {
	var state schema.ResourceState
	switch domain {
	case ResourceStatePending:
		state = schema.ResourceStatePending
	case ResourceStateCreating:
		state = schema.ResourceStateCreating
	case ResourceStateActive:
		state = schema.ResourceStateActive
	case ResourceStateUpdating:
		state = schema.ResourceStateUpdating
	case ResourceStateDeleting:
		state = schema.ResourceStateDeleting
	case ResourceStateSuspended:
		state = schema.ResourceStateSuspended
	case ResourceStateError:
		state = schema.ResourceStateError
	default:
		state = ""
	}
	return state
}

// mapConditionsInStatusDomainToAPI maps StatusDomain.Conditions to a slice of schema.StatusCondition.
func mapConditionsInStatusDomainToAPI(domain StatusDomain) []schema.StatusCondition {
	conditions := make([]schema.StatusCondition, len(domain.Conditions))
	for i, cond := range domain.Conditions {
		conditions[i] = mapConditionDomainToAPI(cond)
	}
	return conditions
}

// mapConditionDomainToAPI maps a StatusConditionDomain to a schema.StatusCondition.
func mapConditionDomainToAPI(domain StatusConditionDomain) schema.StatusCondition {
	return schema.StatusCondition{
		Type:             domain.Type,
		State:            mapResourceStateDomainToAPI(domain.State),
		LastTransitionAt: domain.LastTransitionAt,
		Reason:           domain.Reason,
		Message:          domain.Message,
	}
}
