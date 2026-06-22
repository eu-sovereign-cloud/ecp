package domain

import (
	"time"
)

// ResourceStateDomain represents the current phase of a resource lifecycle.
type ResourceStateDomain string

const (
	ResourceStatePending  ResourceStateDomain = "pending"
	ResourceStateCreating ResourceStateDomain = "creating"
	ResourceStateActive   ResourceStateDomain = "active"
	ResourceStateUpdating ResourceStateDomain = "updating"
	ResourceStateDeleting ResourceStateDomain = "deleting"
	ResourceStateError    ResourceStateDomain = "error"
)

var DefaultPendingCondition = StatusConditionDomain{
	State:   ResourceStatePending,
	Message: "resource is pending",
	Reason:  "Pending",
	Type:    "Pending",
}

// StatusDomain represents the common status attributes of a regional resource. Cannot be directly mapped to schema.Status,
// since <Resource>Status does not embed schema.Status. This is purely for reducing code duplication in regional resource domains.
type StatusDomain struct {
	State      ResourceStateDomain
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
	// Occurrences of condition in consecutive order
	Occurrences int
}

// PushCondition appends condition to StatusDomain.Conditions and mirrors its
// State onto StatusDomain.State, allocating Conditions if needed.
// If condition equals the most recent entry, only its LastTransitionAt
// is updated; no new entry is appended.
func (s *StatusDomain) PushCondition(condition StatusConditionDomain) {
	if s == nil {
		return
	}

	if s.Conditions == nil {
		s.Conditions = []StatusConditionDomain{}
	}

	if condition.LastTransitionAt.IsZero() {
		condition.LastTransitionAt = time.Now()
	}

	prevCondition := s.PeekConditions()
	if prevCondition == nil {
		// ensure that the condition.Occurrences field is initialized to 1 if the condition has not occurred previously
		condition.Occurrences = 1
		s.Conditions = append([]StatusConditionDomain{condition}, s.Conditions...)
		s.State = condition.State
		return
	}

	if EqualStatusConditionDomains(*prevCondition, condition) {
		prevCondition.LastTransitionAt = condition.LastTransitionAt
		prevCondition.Occurrences++
		return
	}

	// ensure that the condition.Occurrences field is initialized to 1 if the condition has not occurred previously
	condition.Occurrences = 1
	s.Conditions = append([]StatusConditionDomain{condition}, s.Conditions...)
	s.State = condition.State
}

// PopCondition removes the oldest (tail) entry from StatusDomain.Conditions.
// It is a no-op when the slice is empty or the receiver is nil.
func (s *StatusDomain) PopCondition() {
	if s == nil || len(s.Conditions) == 0 {
		return
	}

	s.Conditions = s.Conditions[:len(s.Conditions)-1]
}

// PeekConditions returns a pointer to the most recent (head) entry in the StatusDomain.Conditions.
// If there is no Status or no Conditions in the Status, the function will always
// return nil.
func (s *StatusDomain) PeekConditions() *StatusConditionDomain {
	if s == nil || len(s.Conditions) == 0 {
		return nil
	}

	return &s.Conditions[0]
}

// EqualStatusConditionDomains compares two StatusConditionDomains. StatusConditionDomain.LastTransitionAt or
// StatusConditionDomain.Occurrences are not used to compare the two objects.
func EqualStatusConditionDomains(a, b StatusConditionDomain) bool {
	if a.State != b.State {
		return false
	}

	if a.Message != b.Message {
		return false
	}

	if a.Type != b.Type {
		return false
	}

	if a.Reason != b.Reason {
		return false
	}

	return true
}
