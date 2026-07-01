package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	rest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	authnport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authn"
)

// NewAuthentication returns an HTTP middleware that validates the bearer token
// carried by the incoming request's Authorization header.
//
// On success the resolved [authnport.Identity] is stored in the request context
// (retrievable via [IdentityFromContext]) and the next handler is called.
//
// On credential failure (missing/malformed header or invalid token) it writes
// an RFC 7807 401 Unauthorized response.
//
// On technical/infrastructure failure from the authenticator (error wrapping
// [kernel.ErrInternal] or [kernel.ErrUnavailable]) it logs the error server-side
// and writes an RFC 7807 500 Internal Server Error response, so infrastructure
// outages are never silently disguised as authentication failures.
//
// The bearer token format is implementation-specific and determined by the
// injected [authnport.Authenticator]. For the Dummy authenticator used during
// development the token is a base64-encoded JSON object:
//
//	{"username":"alice","password":"secret","roles":["admin-role"]}
//
// WARNING: the Dummy authenticator is for development and testing only.
// Roles are self-asserted by the client and are not cryptographically verified.
func NewAuthentication(a authnport.Authenticator, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				rest.WriteErrorResponse(w, r, log, kernel.ErrUnauthorized)
				return
			}

			identity, err := a.Authenticate(r.Context(), token)
			if err != nil {
				// Distinguish a technical/infrastructure failure from a credential failure.
				// ErrInternal and ErrUnavailable represent problems on the server side;
				// any other error indicates a bad token (credentials) from the client.
				if errors.Is(err, kernel.ErrInternal) || errors.Is(err, kernel.ErrUnavailable) {
					log.ErrorContext(r.Context(), "authn: technical failure authenticating request",
						slog.Any("error", err))
					rest.WriteErrorResponse(w, r, log, kernel.ErrInternal)
				} else {
					rest.WriteErrorResponse(w, r, log, kernel.ErrUnauthorized)
				}
				return
			}

			next.ServeHTTP(w, r.WithContext(contextWithIdentity(r.Context(), identity)))
		})
	}
}

// bearerToken extracts the raw token string from the Authorization header.
// Returns ("", false) when the header is absent or not in "Bearer <token>" form.
func bearerToken(r *http.Request) (string, bool) {
	v := r.Header.Get("Authorization")
	if v == "" {
		return "", false
	}
	parts := strings.SplitN(v, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}
