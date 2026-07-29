// Package middleware provides HTTP middleware abstractions for the ECP gateway.
//
// The middleware chain runs after the HTTP router has matched the request
// (using Go 1.22+ http.ServeMux method-prefixed patterns) so that path values
// such as {tenant} and {workspace} are available via r.PathValue.
//
// Execution order inside oapi-codegen's StdHTTPServerOptions.Middlewares is
// reversed: the last element wraps outermost. Use middleware.Chain to build
// the slice in logical execution order (authentication first, authorization second).
package middleware

import (
	"context"

	authnport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authn"
)

// identityContextKey is the unexported type used as the context key for the
// authenticated Identity. Using a private type prevents key collisions from
// third-party packages.
type identityContextKey struct{}

// contextWithIdentity stores the resolved Identity in the request context.
func contextWithIdentity(ctx context.Context, id *authnport.Identity) context.Context {
	return context.WithValue(ctx, identityContextKey{}, id)
}

// IdentityFromContext retrieves the authenticated Identity stored by the
// authentication middleware. It returns (nil, false) when the middleware has not
// run or the request was not authenticated.
func IdentityFromContext(ctx context.Context) (*authnport.Identity, bool) {
	id, ok := ctx.Value(identityContextKey{}).(*authnport.Identity)
	return id, ok && id != nil
}
