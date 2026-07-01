// Package authz defines the authorization port for the ECP gateway middleware chain.
//
// Implementations live in the gateway module (e.g. SECA RBAC checker) and are
// injected via constructor arguments so the framework layer stays resource-agnostic.
package authz

import (
	"context"
	"net/http"
)

// Decision represents the explicit outcome of an authorization evaluation.
//
// The zero value is intentionally invalid: an unset or default-constructed Decision
// is neither allowed nor denied. The authorization middleware treats any unrecognised
// value (including zero) as a technical error and fails closed with HTTP 500, so a
// buggy Checker implementation can never accidentally grant access.
type Decision int

const (
	// DecisionAllowed indicates the operation is permitted. The authorization
	// middleware passes the request to the next handler.
	DecisionAllowed Decision = iota + 1
	// DecisionDenied indicates the principal is authenticated but lacks sufficient
	// privileges to perform the requested operation. Maps to HTTP 403.
	DecisionDenied
	// DecisionError indicates the authorization decision could not be reached due
	// to a technical or infrastructure failure (e.g. the RBAC store is unreachable).
	// Maps to HTTP 500. The accompanying error carries diagnostic detail for server-side
	// logging but MUST NOT be forwarded to the client verbatim.
	DecisionError
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
//
// Implementations return an explicit Decision alongside an error:
//   - DecisionAllowed, nil:  the operation is permitted; the middleware calls next.
//   - DecisionDenied, err:   authenticated but insufficient privileges; err MUST carry
//     kernel.ErrForbidden (or wrap it) so the caller can log the correct detail.
//     The middleware always responds with HTTP 403 using the sanitised sentinel.
//   - DecisionError, err:    decision unreachable due to a technical/infrastructure failure;
//     err MUST carry a kernel.KindInternal (or kernel.KindUnavailable) error with diagnostic
//     context. The middleware logs err server-side and responds with HTTP 500 using the
//     sanitised sentinel — it MUST NOT forward err to the client.
//
// Implementations MUST NOT return DecisionAllowed with a non-nil error, and MUST NOT
// return a zero/unrecognised Decision value. The middleware fails closed on any
// unrecognised Decision to ensure a buggy implementation cannot accidentally grant access.
type Checker interface {
	// Authorize evaluates the claim against the configured policy and returns
	// an explicit Decision with a complementary error carrying diagnostic detail.
	Authorize(ctx context.Context, claim AuthorizationClaim) (Decision, error)
}

// ClaimExtractor derives an AuthorizationClaim from the current HTTP request.
// A specific provider name and resource-kind-to-verb mapping are baked into
// each ClaimExtractor when it is constructed (see middleware.SECAClaimExtractor).
type ClaimExtractor func(r *http.Request) (AuthorizationClaim, error)
