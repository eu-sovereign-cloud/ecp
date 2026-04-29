package common

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/generated/types"
)

// Conditioned is the interface that all CRD structs should implement if they have a reconcile loop.
//
// +kubebuilder:object:generate=false
type Conditioned interface {
	// PushCondition appends condition to Status.Conditions and mirrors its
	// State onto Status.State. The status value is allocated if the receiver's
	// Status or Conditions slice is nil. If condition equals the most recent
	// entry, only its LastTransitionAt is updated; no new entry is appended.
	PushCondition(types.StatusCondition)
	// GetConditions returns a pointer to the Status.Conditions slice, or
	// nil if the receiver, its Status or the slice is nil.
	GetConditions() []types.StatusCondition
	// PeekConditions returns a pointer to the most recent StatusCondition in the Status.
	// If there is no Status or no Conditions in the Status, the function will always
	// return nil.
	PeekConditions() *types.StatusCondition
	// PopCondition removes the oldest (head) entry from Status.Conditions.
	// It is a no-op when the slice is empty or the receiver has no status.
	PopCondition()
	// LenConditions returns the length of Status.Conditions slice,
	// or zero if the receiver, its Status, or the slice itself is nil.
	LenConditions() int
}

// ConditionedObject is a Kubernetes client.Object that also exposes the
// Conditioned lifecycle. Generic reconcilers target this interface to drive
// status-condition updates without depending on a concrete CRD type.
//
// +kubebuilder:object:generate=false
type ConditionedObject interface {
	Conditioned
	client.Object
}

// EqualConditions compares two StatusConditions. StatusCondition.LastTransitionAt is
// not used to compare the two objects.
func EqualConditions(a, b types.StatusCondition) bool {
	if a.State != b.State {
		return false
	}

	if a.Type != b.Type {
		return false
	}

	if a.Reason != b.Reason {
		return false
	}

	if a.Message != b.Message {
		return false
	}

	return true
}
