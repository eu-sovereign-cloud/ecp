package resolver

import (
	"context"

	resolver_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/resolver"
)

// BypassResolveDependenciesFunc is a helper function that bypasses dependency
// resolution and returns the provided object as is.
func BypassResolveDependenciesFunc[T any](ctx context.Context, main T) (T, error) {
	return main, nil
}

var _ resolver_port.ResolveDependenciesFunc[any, any] = BypassResolveDependenciesFunc[any]

// BypassDependencyResolver is a dependency resolver that bypasses resolution
// and returns the provided object as is.
type BypassDependencyResolver[T any] struct{}

var _ resolver_port.DependenciesResolver[any, any] = (*BypassDependencyResolver[any])(nil)

// ResolveDependencies returns the provided object as is, without resolving any
// dependencies.
func (_ *BypassDependencyResolver[T]) ResolveDependencies(ctx context.Context, main T) (T, error) {
	return main, nil
}
