// Package authn defines the authentication port for the ECP gateway middleware chain.
//
// Implementations live in the gateway module (e.g. dummy bearer-token authenticator)
// and are injected via constructor arguments so the framework layer stays resource-agnostic.
package authn

import "context"

// Identity carries the authenticated subject and any optional down-scoping the caller
// asserts in the bearer token.
//
// The subject is matched against SECA RoleAssignment.Subs during authorization. Roles are
// NOT carried by the token: they are resolved entirely from the RoleAssignment and Role
// resources in the caller's tenant namespace, which are managed by the gateway operator.
type Identity struct {
	// Subject is the authenticated principal (e.g. a username or JWT sub claim).
	// The authorization layer matches it against RoleAssignment.Subs.
	Subject string
	// Scope is an optional down-scoping cap asserted by the token. When a dimension is
	// non-empty, the request's corresponding tenant/region/workspace must be listed or the
	// request is denied. An empty Scope imposes no restriction. Down-scoping can only narrow
	// the permissions granted by RBAC; it never grants anything.
	Scope Scope
}

// Scope is an optional down-scoping cap carried by a bearer token. Each dimension is a list
// of permitted values; an empty (nil) list leaves that dimension unconstrained.
type Scope struct {
	// Tenants restricts the token to the listed tenants; empty means any tenant.
	Tenants []string
	// Regions restricts the token to the listed regions; empty means any region.
	Regions []string
	// Workspaces restricts the token to the listed workspaces; empty means any workspace.
	Workspaces []string
}

// Authenticator validates a raw bearer token and returns the caller's Identity.
//
// Three outcome categories are defined:
//   - Success: returns a non-nil Identity with a nil error; the middleware calls next.
//   - Credential failure (absent, malformed, or invalid token): implementations MUST
//     return kernel.ErrUnauthorized (or wrap it) so the middleware responds HTTP 401.
//   - Technical/infrastructure failure (e.g. the identity provider is unreachable):
//     implementations SHOULD return kernel.ErrInternal or kernel.ErrUnavailable (or
//     wrap either kind) so the middleware responds HTTP 500 instead of 401, clearly
//     distinguishing a transient infrastructure fault from a bad-credentials scenario.
type Authenticator interface {
	// Authenticate decodes and validates the raw bearer token string (without the
	// "Bearer " prefix) and returns the resolved Identity on success.
	Authenticate(ctx context.Context, token string) (*Identity, error)
}
