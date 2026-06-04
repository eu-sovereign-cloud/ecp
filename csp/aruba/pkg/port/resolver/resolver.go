package resolver

import "context"

// ResolveDependenciesFunc is a generic function intended to retrieve the
// dependencies for a main entity and return them in a bundle structure.
type ResolveDependenciesFunc[Main, Bundle any] func(ctx context.Context, main Main) (Bundle, error)

// DependenciesResolver is an interface for types that are capable to resolve
// dependencies for a main entity and return them in a bundle structure.
type DependenciesResolver[Main, Bundle any] interface {
	// ResolveDependencies is a generic function intended to retrieve the
	// dependencies for a main entity and return them in a bundle structure.
	ResolveDependencies(ctx context.Context, main Main) (Bundle, error)
}
