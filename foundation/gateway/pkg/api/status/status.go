package status

import (
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

// ResourceStateDomainToAPI maps ResourceStateDomain to a schema.ResourceState.
func ResourceStateDomainToAPI(domain regional.ResourceStateDomain) schema.ResourceState {
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
	case regional.ResourceStateError:
		state = schema.ResourceStateError
	default:
		state = ""
	}
	return state
}

// ConditionDomainsToAPI maps StatusDomain.Conditions to a slice of schema.StatusCondition.
func ConditionDomainsToAPI(domains []regional.StatusConditionDomain) []schema.StatusCondition {
	conditions := make([]schema.StatusCondition, len(domains))
	for i, cond := range domains {
		conditions[i] = conditionDomainToAPI(cond)
	}
	return conditions
}

// conditionDomainToAPI maps a StatusConditionDomain to a schema.StatusCondition.
func conditionDomainToAPI(domain regional.StatusConditionDomain) schema.StatusCondition {
	return schema.StatusCondition{
		Type:             domain.Type,
		State:            ResourceStateDomainToAPI(domain.State),
		LastTransitionAt: domain.LastTransitionAt,
		Reason:           domain.Reason,
		Message:          domain.Message,
	}
}
