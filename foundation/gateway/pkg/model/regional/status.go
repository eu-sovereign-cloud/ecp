package regional

import (
	"time"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// StatusDomain represents the common status attributes of a regional resource. Cannot be directly mapped to schema.Status,
// since <Resource>Status does not embed schema.Status. This is purely for reducing code duplication in regional resource domains.
type StatusDomain struct {
	State      *string
	Conditions []StatusConditionDomain
}

// StatusConditionDomain represents a single condition of a regional resource's status.
type StatusConditionDomain struct {
	Type             *string
	State            string
	LastTransitionAt time.Time
	Reason           *string
	Message          *string
}

// mapStateInStatusDomainToAPI maps StatusDomain.State to a *schema.ResourceState.
func mapStateInStatusDomainToAPI(domain StatusDomain) *schema.ResourceState {
	var state *schema.ResourceState
	if domain.State != nil {
		state = (*schema.ResourceState)(domain.State)
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
		State:            schema.ResourceState(domain.State),
		LastTransitionAt: domain.LastTransitionAt,
		Reason:           domain.Reason,
		Message:          domain.Message,
	}
}
