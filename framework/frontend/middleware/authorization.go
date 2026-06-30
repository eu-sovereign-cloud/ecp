package middleware

import (
	"log/slog"
	"net/http"

	rest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
)

// NewAuthorization returns an HTTP middleware that enforces the authorization
// policy described by the injected [authzport.Checker].
//
// This is the "Generic Authorization Middleware" (requirement 2.1): it is
// provider-agnostic and delegates all policy decisions to the Checker. The
// concrete SECA RBAC Checker (requirement 2.2) and the Cached variant
// (requirement 2.3) are implemented in the gateway module.
//
// The middleware:
//  1. Retrieves the [authnport.Identity] injected by [NewAuthentication].
//  2. Builds an [authzport.AuthorizationClaim] by calling extract(r) and
//     merging the identity's Roles into the claim.
//  3. Calls checker.Authorize; on nil it calls next. On any non-nil error it
//     writes an RFC 7807 403 Forbidden response.
//
// NewAuthorization MUST be used after NewAuthentication in the middleware chain
// so that the Identity is already present in the context.
func NewAuthorization(
	checker authzport.Checker,
	extract authzport.ClaimExtractor,
	log *slog.Logger,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := IdentityFromContext(r.Context())
			if !ok {
				// Authentication middleware must run before authorization.
				rest.WriteErrorResponse(w, r, log, kernel.ErrUnauthorized)
				return
			}

			claim, err := extract(r)
			if err != nil {
				rest.WriteErrorResponse(w, r, log, kernel.ErrForbidden)
				return
			}
			claim.Roles = identity.Roles

			if err := checker.Authorize(r.Context(), claim); err != nil {
				rest.WriteErrorResponse(w, r, log, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
