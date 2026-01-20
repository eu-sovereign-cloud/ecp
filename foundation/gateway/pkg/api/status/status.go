package status

import (
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// MapResourceStateDomainToAPI maps ResourceStateDomain to a schema.ResourceState.
func MapResourceStateDomainToAPI(domain regional.ResourceStateDomain) schema.ResourceState {
	var state schema.ResourceState
	switch domain {
	case regional.ResourceStatePending:
		state = schema.ResourceStatePending
	case regional.ResourceStateCreating:
		state = schema.ResourceStateCreating
	case regional.ResourceStateActive:
		state = schema.ResourceStateActive
	case regional.ResourceStateUpdating:
		state = schema.ResourceStateUpdating
	case regional.ResourceStateDeleting:
		state = schema.ResourceStateDeleting
	case regional.ResourceStateSuspended:
		state = schema.ResourceStateSuspended
	case regional.ResourceStateError:
		state = schema.ResourceStateError
	default:
		state = ""
	}
	return state
}

// MapConditionDomainsToAPI maps StatusDomain.Conditions to a slice of schema.StatusCondition.
func MapConditionDomainsToAPI(domains []regional.StatusConditionDomain) []schema.StatusCondition {
	conditions := make([]schema.StatusCondition, len(domains))
	for i, cond := range domains {
		conditions[i] = mapConditionDomainToAPI(cond)
	}
	return conditions
}

// mapConditionDomainToAPI maps a StatusConditionDomain to a schema.StatusCondition.
func mapConditionDomainToAPI(domain regional.StatusConditionDomain) schema.StatusCondition {
	return schema.StatusCondition{
		Type:             &domain.Type,
		State:            MapResourceStateDomainToAPI(domain.State),
		LastTransitionAt: domain.LastTransitionAt,
		Reason:           &domain.Reason,
		Message:          &domain.Message,
	}
}
