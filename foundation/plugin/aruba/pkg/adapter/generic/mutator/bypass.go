package mutator

import (
	mutator_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/mutator"
)

// BypassMutateFunc is a helper function that bypasses mutation and returns nil.
// This is useful for scenarios where no mutation is needed on the target object.
func BypassMutateFunc[Mutable, Params any](mutable Mutable, params Params) error {
	return nil
}

var _ mutator_port.MutateFunc[any, any] = BypassMutateFunc[any, any]

// BypassMutator is a mutator that bypasses mutation and does nothing.
// This is useful for scenarios where no mutation is needed on the target object.
type BypassMutator[Mutable, Params any] struct{}

// Ensure BypassMutator implements the Mutator interface
var _ mutator_port.Mutator[any, any] = (*BypassMutator[any, any])(nil)

// Mutate does nothing and returns nil.
func (_ *BypassMutator[Mutable, Params]) Mutate(mutable Mutable, params Params) error {
	return nil
}
