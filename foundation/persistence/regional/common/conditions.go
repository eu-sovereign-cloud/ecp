package common

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/generated/types"
)

// Conditioned is the interface that all CRD structs should implement if they have a reconcile loop.
//
// +kubebuilder:object:generate=false
type Conditioned interface {
	// PushStatusCondition appends condition to Status.Conditions and mirrors its
	// State onto Status.State. The status value is allocated if the receiver's
	// Status or Conditions slice is nil.
	PushStatusCondition(types.StatusCondition)
	// GetStatusConditions returns a pointer to the Status.Conditions slice, or
	// nil if the receiver, its Status, or the slice itself is nil.
	GetStatusConditions() *[]types.StatusCondition
	// PopStatusCondition removes the oldest (head) entry from Status.Conditions.
	// It is a no-op when the slice is empty or the receiver has no status.
	PopStatusCondition()
	// LenStatusConditions returns the length of Status.Conditions slice,
	// or zero if the receiver, its Status, or the slice itself is nil.
	LenStatusConditions() int
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
