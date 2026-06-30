// Package authz defines the authorization port for the ECP gateway middleware chain.
//
// Implementations live in the gateway module (e.g. SECA RBAC checker) and are
// injected via constructor arguments so the framework layer stays resource-agnostic.
package authz

import (
	"context"
	"net/http"
)

// AuthorizationClaim encapsulates all information needed to make a single
// authorization decision. It is assembled by a ClaimExtractor from the incoming
// HTTP request and the Identity established by the authentication middleware.
type AuthorizationClaim struct {
	// Roles is the list of SECA Role names carried by the authenticated identity.
	// The checker intersects this with RoleAssignment.Roles to find applicable grants.
	Roles []string

	// Provider identifies the SECA provider being accessed (e.g. "seca.compute").
	Provider string
	// Resource is the resource kind path derived from the request URL
	// (e.g. "instances", "networks/subnets"). Used for glob matching against
	// Permission.Resources patterns.
	Resource string
	// Name is the specific resource name from the URL path, or empty for collection
	// operations (e.g. list). Used together with Resource for fine-grained matching.
	Name string
	// Verb is the normalized operation verb derived from the HTTP method and route
	// pattern (e.g. "get", "list", "put", "delete", "post.<action>").
	Verb string

	// Tenant is the tenant identifier extracted from the request path.
	Tenant string
	// Region is the region identifier. Set from config.Singleton().Region() for
	// regional servers; empty for the global server.
	Region string
	// Workspace is the workspace identifier extracted from the request path, or
	// empty for tenant-scoped (non-workspace) resources.
	Workspace string
}

// Checker evaluates whether an AuthorizationClaim is permitted.
// Returning nil means the operation is allowed; any non-nil error denies it.
// Implementations should return kernel.ErrForbidden for policy denials.
type Checker interface {
	// Authorize evaluates the claim against the configured policy and returns
	// nil when the operation is permitted or an error when it is denied.
	Authorize(ctx context.Context, claim AuthorizationClaim) error
}

// ClaimExtractor derives an AuthorizationClaim from the current HTTP request.
// A specific provider name and resource-kind-to-verb mapping are baked into
// each ClaimExtractor when it is constructed (see middleware.SECAClaimExtractor).
type ClaimExtractor func(r *http.Request) (AuthorizationClaim, error)
