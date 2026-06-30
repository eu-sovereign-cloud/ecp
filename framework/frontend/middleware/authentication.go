package middleware

import (
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
// On failure (missing header, malformed token, invalid credentials) it writes
// an RFC 7807 401 Unauthorized response and does NOT call next.
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
				rest.WriteErrorResponse(w, r, log, kernel.ErrUnauthorized)
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
