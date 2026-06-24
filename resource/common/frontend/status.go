package frontend

import (
	"fmt"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// ResourceStateDomainToAPI maps domain.ResourceState to a schema.ResourceState.
func ResourceStateDomainToAPI(d domain.ResourceState) schema.ResourceState {
	var state schema.ResourceState
	switch d {
	case domain.ResourceStatePending:
		state = schema.ResourceStatePending
	case domain.ResourceStateCreating:
		state = schema.ResourceStateCreating
	case domain.ResourceStateActive:
		state = schema.ResourceStateActive
	case domain.ResourceStateUpdating:
		state = schema.ResourceStateUpdating
	case domain.ResourceStateDeleting:
		state = schema.ResourceStateDeleting
	case domain.ResourceStateError:
		state = schema.ResourceStateError
	default:
		state = ""
	}
	return state
}

// ConditionDomainsToAPI maps Status.Conditions to a slice of schema.StatusCondition.
func ConditionDomainsToAPI(domains []domain.StatusCondition) []schema.StatusCondition {
	conditions := make([]schema.StatusCondition, len(domains))
	for i, cond := range domains {
		conditions[i] = conditionDomainToAPI(cond)
	}
	return conditions
}

// conditionDomainToAPI maps a domain.StatusCondition to a schema.StatusCondition.
func conditionDomainToAPI(d domain.StatusCondition) schema.StatusCondition {
	return schema.StatusCondition{
		Type:             d.Type,
		State:            ResourceStateDomainToAPI(d.State),
		LastTransitionAt: d.LastTransitionAt,
		Reason:           d.Reason,
		Message:          fmt.Sprintf("%s (x%d)", d.Message, d.Occurrences),
	}
}
