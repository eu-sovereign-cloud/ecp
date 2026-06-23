// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

package v1

import "sigs.k8s.io/controller-runtime/pkg/client"

// Conditioned is the interface that all CRD structs should implement if they have a reconcile loop.
//
// +kubebuilder:object:generate=false
type Conditioned interface {
	// PushCondition appends condition to Status.Conditions and mirrors its
	// State onto Status.State. The status value is allocated if the receiver's
	// Status or Conditions slice is nil. If condition equals the most recent
	// entry, only its LastTransitionAt is updated; no new entry is appended.
	PushCondition(StatusCondition)
	// GetConditions returns a pointer to the Status.Conditions slice, or
	// nil if the receiver, its Status or the slice is nil.
	GetConditions() []StatusCondition
	// PeekConditions returns a pointer to the most recent StatusCondition in the Status.
	// If there is no Status or no Conditions in the Status, the function will always
	// return nil.
	PeekConditions() *StatusCondition
	// PopCondition removes the oldest (tail) entry from Status.Conditions.
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

// EqualConditions compares two StatusConditions. StatusCondition.LastTransitionAt or StatusCondition.Occurrences are
// not used to compare the two objects.
func EqualConditions(a, b StatusCondition) bool {
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

// CommonData defines the additional common fields that can be set on resources
type CommonData struct {
	// Annotations User-defined key/value pairs that are mutable and can be used to add annotations.
	// The number of annotations is eventually limited by the CSP.
	Annotations Annotations `json:"annotations,omitempty"`

	// Extensions User-defined key/value pairs that are mutable and can be used to add extensions.
	// Extensions are subject to validation by the CSP, and any value that is not accepted will be rejected during admission.
	Extensions map[string]string `json:"extensions,omitempty"`

	// Labels User-defined key/value pairs that are mutable and can be used to
	// organize and categorize resources. We store the keys explicitly in the spec, because the values will be stored
	// directly in the Kubernetes labels.
	Labels []string `json:"labels,omitempty"`
}
