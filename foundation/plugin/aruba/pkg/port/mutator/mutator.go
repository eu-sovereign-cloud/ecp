package mutator

// MutateFunc is a generic function that applies mutations to a mutable entity
// based on provided parameters.
type MutateFunc[Mutable, Params any] func(mutable Mutable, params Params) error

// Mutator is an interface for types that can apply mutations to an entity
// based on provided parameters.
type Mutator[Mutable, Params any] interface {
	// Mutate applies mutations to a mutable entity based on provided
	// parameters.
	Mutate(mutable Mutable, params Params) error
}
