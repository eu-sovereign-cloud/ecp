// Package authn defines the authentication port for the ECP gateway middleware chain.
//
// Implementations live in the gateway module (e.g. dummy bearer-token authenticator)
// and are injected via constructor arguments so the framework layer stays resource-agnostic.
package authn

import "context"

// Identity carries the authenticated subject and the set of role names the subject holds.
// The role names are compared against SECA Role resources during authorization.
type Identity struct {
	// Subject is the authenticated principal (e.g. a username or service account email).
	Subject string
	// Roles is the list of SECA Role names carried by the bearer token.
	// The authorization layer uses this set to check SECA RoleAssignments.
	Roles []string
}

// Authenticator validates a raw bearer token and returns the caller's Identity.
// Implementations must return kernel.ErrUnauthorized when the token is absent,
// malformed, or carries invalid credentials.
type Authenticator interface {
	// Authenticate decodes and validates the raw bearer token string (without the
	// "Bearer " prefix) and returns the resolved Identity on success.
	Authenticate(ctx context.Context, token string) (*Identity, error)
}
