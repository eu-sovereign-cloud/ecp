package frontend

import (
	"fmt"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// ResourceStateToAPI maps domain.ResourceState to a schema.ResourceState.
func ResourceStateToAPI(state domain.ResourceState) schema.ResourceState {
	var out schema.ResourceState
	switch state {
	case domain.ResourceStatePending:
		out = schema.ResourceStatePending
	case domain.ResourceStateCreating:
		out = schema.ResourceStateCreating
	case domain.ResourceStateActive:
		out = schema.ResourceStateActive
	case domain.ResourceStateUpdating:
		out = schema.ResourceStateUpdating
	case domain.ResourceStateDeleting:
		out = schema.ResourceStateDeleting
	case domain.ResourceStateError:
		out = schema.ResourceStateError
	default:
		out = ""
	}
	return out
}

// IPVersionToAPI maps domain.IPVersion to schema.IPVersion.
func IPVersionToAPI(v domain.IPVersion) schema.IPVersion {
	switch v {
	case domain.IPVersionIPv4:
		return schema.IPVersionIPv4
	case domain.IPVersionIPv6:
		return schema.IPVersionIPv6
	default:
		return ""
	}
}

// IPVersionFromAPI maps schema.IPVersion to domain.IPVersion.
func IPVersionFromAPI(v schema.IPVersion) domain.IPVersion {
	switch v {
	case schema.IPVersionIPv4:
		return domain.IPVersionIPv4
	case schema.IPVersionIPv6:
		return domain.IPVersionIPv6
	default:
		return ""
	}
}

// ConditionsToAPI maps Status.Conditions to a slice of schema.StatusCondition.
func ConditionsToAPI(conds []domain.StatusCondition) []schema.StatusCondition {
	conditions := make([]schema.StatusCondition, len(conds))
	for i, cond := range conds {
		conditions[i] = conditionToAPI(cond)
	}
	return conditions
}

// conditionToAPI maps a domain.StatusCondition to a schema.StatusCondition.
func conditionToAPI(c domain.StatusCondition) schema.StatusCondition {
	return schema.StatusCondition{
		Type:             c.Type,
		State:            ResourceStateToAPI(c.State),
		LastTransitionAt: c.LastTransitionAt,
		Reason:           c.Reason,
		Message:          fmt.Sprintf("%s (x%d)", c.Message, c.Occurrences),
	}
}
