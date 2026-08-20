// Package auth provides the opt-in authentication and authorization configuration
// for the ECP gateway.
//
// The middleware chain is DISABLED by default; it is activated only when
// --auth-enabled is set. This allows existing deployments and integration tests
// to operate without a valid bearer token until the feature is rolled out.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"k8s.io/client-go/dynamic"

	middleware "github.com/eu-sovereign-cloud/ecp/framework/frontend/middleware"
	authnport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authn"
	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	gatewayauthn "github.com/eu-sovereign-cloud/ecp/gateway/internal/authn"
	seca "github.com/eu-sovereign-cloud/ecp/gateway/internal/authz/seca"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/metrics"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
)

// Flags holds the parsed command-line values for the auth subsystem.
// Use RegisterFlags to bind these to a cobra command.
type Flags struct {
	// Enabled turns the entire auth chain on; when false, no middlewares are installed
	// and existing deployments are unaffected.
	Enabled bool
	// AuthPlugin selects the authentication plugin to use. Supports "dummy" (static username→password map) and "jwt" (standard signed JWTs). Default "dummy".
	AuthPlugin string
	// JwtSigningMethod is the expected JWT signing method (e.g. "ES256") when AuthPlugin is "jwt". Required when AuthPlugin is "jwt".
	JwtSigningMethod string
	// JwtSecretFile is the path to a file holding the verification key for JWTs when AuthPlugin is "jwt". Required when AuthPlugin is "jwt". For HS* the file content is the raw HMAC secret; for all other methods it is a PEM-encoded PKIX public key.
	JwtSecretFile string
	// JwtIssuer is the expected "iss" claim. Empty accepts any issuer; when set, a token
	// with a different or missing issuer is rejected.
	JwtIssuer string
	// JwtAudience is the expected "aud" claim. Empty accepts any audience; when set, a
	// token whose audience does not include it (or omits the claim) is rejected.
	JwtAudience string
	// DummyUsersFile is the path to a JSON file containing username→password pairs.
	// Required when Enabled is true. Example file content: {"alice":"s3cr3t","bob":"p@ss"}
	DummyUsersFile string
	// AuthzCache enables the informer-backed CachedChecker instead of the per-request
	// reader-backed Checker. When true, Build also requires a non-nil dynClient.
	AuthzCache bool
	// AuthzEnabled controls whether the authorization middleware is installed alongside
	// the authentication middleware. Requires Enabled to be true. When false, requests
	// pass through authn only (authn-only mode); authorization is skipped entirely.
	// Default true when auth is enabled.
	AuthzEnabled bool
	// AuthzSkipProviders lists provider IDs (e.g. "seca.region") whose routes skip the
	// authorization middleware even when AuthzEnabled is true: callers must still
	// authenticate, but no RBAC check or token down-scoping is applied. Meant for
	// resources that tenant-scoped RBAC cannot govern — by default the region catalog,
	// which is tenant-less by spec.
	AuthzSkipProviders []string
}

// RegisterFlags adds auth-related flags to the given cobra command.
func RegisterFlags(cmd *cobra.Command, f *Flags) {
	cmd.Flags().BoolVar(&f.Enabled, "auth-enabled", false,
		"Enable bearer-token authentication and SECA RBAC authorization (disabled by default)")
	cmd.Flags().StringVar(&f.AuthPlugin, "auth-plugin", "dummy", "Authentication plugin to use (one of: dummy, jwt)")
	cmd.Flags().StringVar(&f.DummyUsersFile, "dummy-auth-users", "",
		"Path to a JSON file mapping username→password for the Dummy authenticator "+
			"(required when --auth-enabled is set)")
	cmd.Flags().StringVar(&f.JwtSigningMethod, "jwt-signing-method", "ES256", "Expected JWT signing method when --auth-plugin is 'jwt' (required when --auth-plugin is 'jwt')")
	cmd.Flags().StringVar(&f.JwtSecretFile, "jwt-secret", "", "Path to a file containing the JWT verification key: the raw HMAC secret for HS*, a PEM public key otherwise (required when --auth-plugin is 'jwt')")
	cmd.Flags().StringVar(&f.JwtIssuer, "jwt-issuer", "", "Expected JWT 'iss' claim; when set, tokens from another issuer (or without the claim) are rejected (empty accepts any issuer)")
	cmd.Flags().StringVar(&f.JwtAudience, "jwt-audience", "", "Expected JWT 'aud' claim; when set, tokens for another audience (or without the claim) are rejected (empty accepts any audience)")
	cmd.Flags().BoolVar(&f.AuthzCache, "authz-cache", false,
		"Use the informer-backed CachedChecker instead of the per-request RBAC checker "+
			"(requires --auth-enabled; reduces API-server load on hot paths)")
	cmd.Flags().BoolVar(&f.AuthzEnabled, "authz-enabled", true,
		"Install the authorization middleware; requires --auth-enabled; "+
			"set false for authn-only mode (identity is verified but no RBAC check is performed)")
	cmd.Flags().StringSliceVar(&f.AuthzSkipProviders, "authz-skip-providers", []string{"seca.region"},
		"Comma-separated provider IDs whose routes skip the authorization middleware "+
			"(authn-only; no RBAC check or token down-scoping); the region catalog is "+
			"tenant-less by spec, so seca.region is skipped by default")
}

// Build constructs the Authenticator and Checker from the provided flags and readers.
// Returns (nil, nil, nil) when auth is disabled — callers may check authenticator == nil
// to skip middleware wiring.
//
// When flags.AuthzCache is true, a CachedChecker is built using dynClient (which must be
// non-nil). The caller is responsible for calling CachedChecker.Start before the server
// starts serving requests.
//
// Returns an error if --auth-enabled is true but the users file is missing or invalid.
func Build(
	flags *Flags,
	dynClient dynamic.Interface,
	roleReader persistence.ReaderRepo[*roledom.Role],
	assignmentReader persistence.ReaderRepo[*radom.RoleAssignment],
	log *slog.Logger,
) (authnport.Authenticator, authzport.Checker, error) {
	if !flags.Enabled {
		return nil, nil, nil
	}

	authenticator, err := buildAuthenticator(flags)
	if err != nil {
		return nil, nil, fmt.Errorf("build authenticator: %w", err)
	}

	if !flags.AuthzEnabled {
		// Authn-only mode: identity is verified but no RBAC check is performed.
		return authenticator, nil, nil
	}

	if flags.AuthzCache {
		if dynClient == nil {
			return nil, nil, fmt.Errorf("--authz-cache requires a dynamic Kubernetes client")
		}
		checker := seca.NewCachedChecker(dynClient, log)
		return authenticator, metrics.NewInstrumentedChecker(checker, "cached"), nil
	}

	checker := seca.NewChecker(roleReader, assignmentReader, log)
	return authenticator, metrics.NewInstrumentedChecker(checker, "direct"), nil
}

// AuthnMiddleware returns the authentication middleware for the given Authenticator,
// or nil when the authenticator is nil (auth disabled).
func AuthnMiddleware(authenticator authnport.Authenticator, log *slog.Logger) func(http.Handler) http.Handler {
	if authenticator == nil {
		return nil
	}
	return middleware.NewAuthentication(authenticator, log)
}

// AuthzMiddleware returns an authorization middleware bound to the given provider and
// base URL, or nil when the checker is nil (auth disabled).
//
// Each provider registration calls this with its own provider ID and base URL so
// that the claim extractor's provider field is correctly set per provider.
//
// Example:
//
//	authzMW := auth.AuthzMiddleware(checker, "seca.network", "/providers/seca.network", log)
//	opts.Middlewares = middleware.Chain[sdknetworkapi.MiddlewareFunc](authnMW, authzMW)
func AuthzMiddleware(checker authzport.Checker, provider, baseURL string, log *slog.Logger) func(http.Handler) http.Handler {
	if checker == nil {
		return nil
	}
	return middleware.NewAuthorization(checker, middleware.SECAClaimExtractor(provider, baseURL), log)
}

// ProviderMWs returns the typed middleware slice for a provider when auth is enabled,
// or nil when auth is disabled (authenticator == nil).
//
// This is the primary wiring helper. Use it inside HandlerWithOptions:
//
//	authv1.HandlerWithOptions(handler, authv1.StdHTTPServerOptions{
//	    Middlewares: auth.ProviderMWs[authv1.MiddlewareFunc](
//	        flags, authenticator, checker,
//	        "seca.authorization", roledom.AuthorizationBaseURL,
//	        log,
//	    ),
//	})
//
// A provider listed in flags.AuthzSkipProviders gets the authn-only chain even when
// the checker is non-nil: its routes are authenticated but never authorized.
//
// Returning nil preserves the existing behavior (no-op mux, no bearer check) when
// --auth-enabled is not set.
func ProviderMWs[M ~func(http.Handler) http.Handler](
	flags *Flags,
	authenticator authnport.Authenticator,
	checker authzport.Checker,
	provider, baseURL string,
	log *slog.Logger,
) []M {
	if authenticator == nil {
		return nil
	}
	metricsMW := metrics.Middleware(provider)
	authnMW := middleware.NewAuthentication(authenticator, log)
	if checker == nil {
		// Authn-only mode: skip authorization middleware.
		return middleware.Chain[M](metricsMW, authnMW)
	}
	if flags.authzSkipped(provider) {
		log.Info("authorization middleware skipped for provider by configuration",
			slog.String("provider", provider))
		return middleware.Chain[M](metricsMW, authnMW)
	}
	authzMW := middleware.NewAuthorization(checker, middleware.SECAClaimExtractor(provider, baseURL), log)
	// metrics.Middleware is the first argument so Chain places it outermost (Chain reverses).
	return middleware.Chain[M](metricsMW, authnMW, authzMW)
}

// authzSkipped reports whether the provider is listed in AuthzSkipProviders.
func (f *Flags) authzSkipped(provider string) bool {
	return f != nil && slices.Contains(f.AuthzSkipProviders, provider)
}

// StartChecker starts the checker's background cache goroutines if it implements the
// optional Starter interface (i.e. it is a CachedChecker). It is a no-op for the
// plain Checker and when checker is nil (auth disabled).
//
// The wrapping InstrumentedChecker forwards Start, so this type-assertion succeeds for
// both the direct and cached checkers; the actual "starting informer-backed checker"
// log is emitted by CachedChecker.Start so it only fires when the cache is truly used.
func StartChecker(ctx context.Context, checker authzport.Checker, log *slog.Logger) error {
	type starter interface{ Start(context.Context) error }
	if s, ok := checker.(starter); ok {
		return s.Start(ctx)
	}
	return nil
}

// buildAuthenticator loads the Dummy authenticator from the configured users file.
func buildAuthenticator(flags *Flags) (authnport.Authenticator, error) {
	switch flags.AuthPlugin {
	case "dummy":
		if flags.DummyUsersFile == "" {
			return nil, fmt.Errorf("--dummy-auth-users must be set when --auth-enabled is true")
		}
		data, err := os.ReadFile(flags.DummyUsersFile)
		if err != nil {
			return nil, fmt.Errorf("read dummy users file %q: %w", flags.DummyUsersFile, err)
		}
		var users map[string]string
		if err := json.Unmarshal(data, &users); err != nil {
			return nil, fmt.Errorf("parse dummy users file %q: %w", flags.DummyUsersFile, err)
		}
		return gatewayauthn.NewDummyAuthenticator(users), nil
	case "jwt":
		if flags.JwtSecretFile == "" || flags.JwtSigningMethod == "" {
			return nil, fmt.Errorf("--jwt-secret and --jwt-signing-method must be set when --auth-plugin is 'jwt'")
		}
		data, err := os.ReadFile(flags.JwtSecretFile)
		if err != nil {
			return nil, fmt.Errorf("read JWT secret file %q: %w", flags.JwtSecretFile, err)
		}
		key, err := gatewayauthn.ParseVerifyKey(flags.JwtSigningMethod, data)
		if err != nil {
			return nil, fmt.Errorf("parse JWT key from %q: %w", flags.JwtSecretFile, err)
		}
		return gatewayauthn.NewJWTAuthenticator(key, flags.JwtSigningMethod, flags.JwtIssuer, flags.JwtAudience), nil
	}
	return nil, fmt.Errorf("unknown auth plugin %q", flags.AuthPlugin)
}
