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
	// DummyUsersFile is the path to a JSON file containing username→password pairs.
	// Required when Enabled is true. Example file content: {"alice":"s3cr3t","bob":"p@ss"}
	DummyUsersFile string
	// AuthzCache enables the informer-backed CachedChecker instead of the per-request
	// reader-backed Checker. When true, Build also requires a non-nil dynClient.
	AuthzCache bool
}

// RegisterFlags adds auth-related flags to the given cobra command.
func RegisterFlags(cmd *cobra.Command, f *Flags) {
	cmd.Flags().BoolVar(&f.Enabled, "auth-enabled", false,
		"Enable bearer-token authentication and SECA RBAC authorization (disabled by default)")
	cmd.Flags().StringVar(&f.DummyUsersFile, "dummy-auth-users", "",
		"Path to a JSON file mapping username→password for the Dummy authenticator "+
			"(required when --auth-enabled is set)")
	cmd.Flags().BoolVar(&f.AuthzCache, "authz-cache", false,
		"Use the informer-backed CachedChecker instead of the per-request RBAC checker "+
			"(requires --auth-enabled; reduces API-server load on hot paths)")
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
//	        authenticator, checker,
//	        "seca.authorization", roledom.AuthorizationBaseURL,
//	        log,
//	    ),
//	})
//
// Returning nil preserves the existing behavior (no-op mux, no bearer check) when
// --auth-enabled is not set.
func ProviderMWs[M ~func(http.Handler) http.Handler](
	authenticator authnport.Authenticator,
	checker authzport.Checker,
	provider, baseURL string,
	log *slog.Logger,
) []M {
	if authenticator == nil {
		return nil
	}
	authnMW := middleware.NewAuthentication(authenticator, log)
	authzMW := middleware.NewAuthorization(checker, middleware.SECAClaimExtractor(provider, baseURL), log)
	return middleware.Chain[M](authnMW, authzMW)
}

// StartChecker starts the checker's background cache goroutines if it implements the
// optional Starter interface (i.e. it is a CachedChecker). It is a no-op for the
// plain Checker and when checker is nil (auth disabled).
func StartChecker(ctx context.Context, checker authzport.Checker, log *slog.Logger) error {
	type starter interface{ Start(context.Context) error }
	if s, ok := checker.(starter); ok {
		log.Info("authz cache: starting informer-backed checker")
		return s.Start(ctx)
	}
	return nil
}

// buildAuthenticator loads the Dummy authenticator from the configured users file.
func buildAuthenticator(flags *Flags) (*gatewayauthn.DummyAuthenticator, error) {
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
}
