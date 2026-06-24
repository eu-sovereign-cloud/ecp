package domain

import (
	"time"
)

// ResourceState represents the current phase of a resource lifecycle.
type ResourceState string

const (
	ResourceStatePending  ResourceState = "pending"
	ResourceStateCreating ResourceState = "creating"
	ResourceStateActive   ResourceState = "active"
	ResourceStateUpdating ResourceState = "updating"
	ResourceStateDeleting ResourceState = "deleting"
	ResourceStateError    ResourceState = "error"
)

var DefaultPendingCondition = StatusCondition{
	State:   ResourceStatePending,
	Message: "resource is pending",
	Reason:  "Pending",
	Type:    "Pending",
}

// Status represents the common status attributes of a regional resource. Cannot be directly mapped to schema.Status,
// since <Resource>Status does not embed schema.Status. This is purely for reducing code duplication in regional resource domains.
type Status struct {
	State      ResourceState
	Conditions []StatusCondition
}

// StatusCondition describes a single state condition of a regional resource's status at a certain point in time.
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
	// Occurrences of condition in consecutive order
	Occurrences int
}

// PushCondition appends condition to Status.Conditions and mirrors its
// State onto Status.State, allocating Conditions if needed.
// If condition equals the most recent entry, only its LastTransitionAt
// is updated; no new entry is appended.
func (s *Status) PushCondition(condition StatusCondition) {
	if s == nil {
		return
	}

	if s.Conditions == nil {
		s.Conditions = []StatusCondition{}
	}

	if condition.LastTransitionAt.IsZero() {
		condition.LastTransitionAt = time.Now()
	}

	prevCondition := s.PeekConditions()
	if prevCondition == nil {
		// ensure that the condition.Occurrences field is initialized to 1 if the condition has not occurred previously
		condition.Occurrences = 1
		s.Conditions = append([]StatusCondition{condition}, s.Conditions...)
		s.State = condition.State
		return
	}

	if EqualStatusConditions(*prevCondition, condition) {
		prevCondition.LastTransitionAt = condition.LastTransitionAt
		prevCondition.Occurrences++
		return
	}

	// ensure that the condition.Occurrences field is initialized to 1 if the condition has not occurred previously
	condition.Occurrences = 1
	s.Conditions = append([]StatusCondition{condition}, s.Conditions...)
	s.State = condition.State
}

// PopCondition removes the oldest (tail) entry from Status.Conditions.
// It is a no-op when the slice is empty or the receiver is nil.
func (s *Status) PopCondition() {
	if s == nil || len(s.Conditions) == 0 {
		return
	}

	s.Conditions = s.Conditions[:len(s.Conditions)-1]
}

// PeekConditions returns a pointer to the most recent (head) entry in the Status.Conditions.
// If there is no Status or no Conditions in the Status, the function will always
// return nil.
func (s *Status) PeekConditions() *StatusCondition {
	if s == nil || len(s.Conditions) == 0 {
		return nil
	}

	return &s.Conditions[0]
}

// EqualStatusConditions compares two StatusConditions. StatusCondition.LastTransitionAt or
// StatusCondition.Occurrences are not used to compare the two objects.
func EqualStatusConditions(a, b StatusCondition) bool {
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
