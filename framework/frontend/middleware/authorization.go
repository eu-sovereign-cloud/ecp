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
//  2. Builds an [authzport.AuthorizationClaim] by calling extract(r) and merging
//     the identity's Subject and Roles into the claim. A claim-extraction error
//     is treated as a technical fault and yields HTTP 500.
//  3. Calls checker.Authorize and branches on the returned [authzport.Decision]:
//     [authzport.DecisionAllowed] → calls next handler (HTTP 2xx).
//     [authzport.DecisionDenied]  → writes RFC 7807 HTTP 403 Forbidden.
//     [authzport.DecisionError]   → logs the detailed error server-side and writes
//     RFC 7807 HTTP 500. Any unrecognised Decision (including the zero value) also
//     yields HTTP 500 so the middleware fails closed.
//
// NewAuthorization MUST be used after NewAuthentication in the middleware chain
// so that the Identity is already present in the context. The middleware merges
// both identity.Roles and identity.Subject into the claim before invoking the
// checker.
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
				// Claim-extraction failures are technical faults (e.g. unrecognised
				// URL pattern), not policy denials — return 500, not 403.
				log.ErrorContext(r.Context(), "authz: failed to extract authorization claim",
					slog.Any("error", err))
				rest.WriteErrorResponse(w, r, log, kernel.ErrInternal)
				return
			}
			claim.Subject = identity.Subject
			claim.Roles = identity.Roles

			decision, decErr := checker.Authorize(r.Context(), claim)
			switch decision {
			case authzport.DecisionAllowed:
				next.ServeHTTP(w, r)
			case authzport.DecisionDenied:
				rest.WriteErrorResponse(w, r, log, kernel.ErrForbidden)
			case authzport.DecisionError:
				// Log the detailed diagnostic error server-side; respond with the
				// sanitised sentinel to avoid leaking internal infrastructure details.
				log.ErrorContext(r.Context(), "authz: technical failure evaluating authorization",
					slog.Any("error", decErr))
				rest.WriteErrorResponse(w, r, log, kernel.ErrInternal)
			default:
				// Zero or unrecognised Decision — fail closed.
				log.ErrorContext(r.Context(), "authz: checker returned unrecognised decision — failing closed",
					slog.Int("decision", int(decision)), slog.Any("error", decErr))
				rest.WriteErrorResponse(w, r, log, kernel.ErrInternal)
			}
		})
	}
}
