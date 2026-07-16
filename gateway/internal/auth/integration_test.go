// Package auth_test contains in-process integration tests that exercise the
// full token → authn → authz → handler pipeline assembled by this package.
package auth_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"

	"github.com/eu-sovereign-cloud/ecp/framework/frontend/middleware"
	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	authnport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authn"
	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/auth"
	gatewayauthn "github.com/eu-sovereign-cloud/ecp/gateway/internal/authn"
	seca "github.com/eu-sovereign-cloud/ecp/gateway/internal/authz/seca"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	commondom "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// ── test doubles ─────────────────────────────────────────────────────────────

// checkerFunc is a test-only implementation of authzport.Checker backed by a
// function literal.
type checkerFunc func(ctx context.Context, claim authzport.AuthorizationClaim) (authzport.Decision, error)

func (f checkerFunc) Authorize(ctx context.Context, claim authzport.AuthorizationClaim) (authzport.Decision, error) {
	return f(ctx, claim)
}

// allowChecker always permits.
var allowChecker checkerFunc = func(_ context.Context, _ authzport.AuthorizationClaim) (authzport.Decision, error) {
	return authzport.DecisionAllowed, nil
}

// denyChecker always forbids.
var denyChecker checkerFunc = func(_ context.Context, _ authzport.AuthorizationClaim) (authzport.Decision, error) {
	return authzport.DecisionDenied, kernel.ErrForbidden
}

// errorChecker simulates a technical failure (e.g. the RBAC store is unreachable).
var errorChecker checkerFunc = func(_ context.Context, _ authzport.AuthorizationClaim) (authzport.Decision, error) {
	return authzport.DecisionError, kernel.ErrInternal
}

// rbacChecker delegates to the real seca.Evaluate so tests exercise the actual
// authorization algorithm (including the token down-scope gate) end-to-end.
func rbacChecker(roles map[string]*roledom.Role, assignments []*radom.RoleAssignment) checkerFunc {
	return func(_ context.Context, claim authzport.AuthorizationClaim) (authzport.Decision, error) {
		if seca.Evaluate(claim, roles, assignments) {
			return authzport.DecisionAllowed, nil
		}
		return authzport.DecisionDenied, kernel.ErrForbidden
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// bearerToken encodes a Dummy-auth bearer token from the given credentials and an
// optional token scope (down-scope cap). The wire scope reuses [resource.TokenScope] and
// its json tags. Roles are never carried by the token.
func bearerToken(username, password string, scope *resource.TokenScope) string {
	type payload struct {
		Username string               `json:"username"`
		Password string               `json:"password"`
		Scope    *resource.TokenScope `json:"scope,omitempty"`
	}
	b, err := json.Marshal(payload{Username: username, Password: password, Scope: scope})
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// okHandler is the leaf handler that records a successful pass-through.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// fixedExtractor always returns the same AuthorizationClaim (no mux involved).
var fixedExtractor authzport.ClaimExtractor = func(_ *http.Request) (authzport.AuthorizationClaim, error) {
	return authzport.AuthorizationClaim{
		Provider:  "seca.compute",
		Resource:  "instances",
		Verb:      "list",
		Tenant:    "t1",
		Region:    "",
		Workspace: "",
	}, nil
}

// buildChain returns an HTTP handler with the authn+authz middlewares applied.
func buildChain(a authnport.Authenticator, c authzport.Checker) http.Handler {
	log := discardLog()
	authnMW := middleware.NewAuthentication(a, log)
	authzMW := middleware.NewAuthorization(c, fixedExtractor, log)
	return authnMW(authzMW(okHandler))
}

// ── integration tests ─────────────────────────────────────────────────────────

// TestIntegration_ValidToken_Allowed is the happy path:
// valid bearer token + RBAC allows → 200.
func TestIntegration_ValidToken_Allowed(t *testing.T) {
	t.Parallel()

	a := gatewayauthn.NewDummyAuthenticator(map[string]string{"alice": "s3cr3t"})
	h := buildChain(a, allowChecker)

	req := httptest.NewRequest(http.MethodGet, "/instances", nil)
	req.Header.Set("Authorization", "Bearer "+bearerToken("alice", "s3cr3t", nil))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestIntegration_MissingToken returns 401.
func TestIntegration_MissingToken(t *testing.T) {
	t.Parallel()

	a := gatewayauthn.NewDummyAuthenticator(map[string]string{"alice": "s3cr3t"})
	h := buildChain(a, allowChecker)

	req := httptest.NewRequest(http.MethodGet, "/instances", nil)
	// no Authorization header
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

// TestIntegration_WrongPassword returns 401 (invalid token).
func TestIntegration_WrongPassword(t *testing.T) {
	t.Parallel()

	a := gatewayauthn.NewDummyAuthenticator(map[string]string{"alice": "s3cr3t"})
	h := buildChain(a, allowChecker)

	req := httptest.NewRequest(http.MethodGet, "/instances", nil)
	req.Header.Set("Authorization", "Bearer "+bearerToken("alice", "wrongpass", nil))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

// TestIntegration_ValidToken_Denied returns 403 when the checker denies.
func TestIntegration_ValidToken_Denied(t *testing.T) {
	t.Parallel()

	a := gatewayauthn.NewDummyAuthenticator(map[string]string{"alice": "s3cr3t"})
	h := buildChain(a, denyChecker)

	req := httptest.NewRequest(http.MethodGet, "/instances", nil)
	req.Header.Set("Authorization", "Bearer "+bearerToken("alice", "s3cr3t", nil))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", w.Code)
	}
}

// TestIntegration_CheckerTechnicalError verifies that a technical failure in the
// authorization checker (e.g. the RBAC store unreachable) yields HTTP 500, not a
// 403 denial — confirming that technical errors are never disguised as policy denials.
func TestIntegration_CheckerTechnicalError(t *testing.T) {
	t.Parallel()

	a := gatewayauthn.NewDummyAuthenticator(map[string]string{"alice": "s3cr3t"})
	h := buildChain(a, errorChecker)

	req := httptest.NewRequest(http.MethodGet, "/instances", nil)
	req.Header.Set("Authorization", "Bearer "+bearerToken("alice", "s3cr3t", nil))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d — technical checker errors must not be disguised as 403", w.Code)
	}
}

// TestIntegration_AuthnOnly verifies that when the checker is nil (--authz-enabled=false),
// valid credentials grant access without an RBAC decision. The handler is reached
// with a 200 and no authzport.Checker is consulted.
func TestIntegration_AuthnOnly(t *testing.T) {
	t.Parallel()

	a := gatewayauthn.NewDummyAuthenticator(map[string]string{"alice": "s3cr3t"})
	log := discardLog()
	authnMW := middleware.NewAuthentication(a, log)
	// No authzMW — checker nil simulates --authz-enabled=false.
	h := authnMW(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/instances", nil)
	req.Header.Set("Authorization", "Bearer "+bearerToken("alice", "s3cr3t", nil))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200 in authn-only mode, got %d", w.Code)
	}
}

// wrapMWs applies a ProviderMWs slice the way oapi-codegen does: wrapping from the
// start of the slice inward, so the LAST element ends up outermost (runs first).
func wrapMWs[M ~func(http.Handler) http.Handler](mws []M, h http.Handler) http.Handler {
	for _, mw := range mws {
		h = mw(h)
	}
	return h
}

// TestIntegration_AuthzSkipProviders verifies the per-provider authorization skip
// (--authz-skip-providers): a listed provider is authenticated but never authorized —
// a checker that would deny everything cannot block it — while an unlisted provider
// keeps the full authn+authz chain. Authentication is still enforced on the skipped
// provider (missing token → 401).
func TestIntegration_AuthzSkipProviders(t *testing.T) {
	t.Parallel()

	a := gatewayauthn.NewDummyAuthenticator(map[string]string{"alice": "s3cr3t"})
	flags := &auth.Flags{Enabled: true, AuthzEnabled: true, AuthzSkipProviders: []string{"seca.region"}}
	log := discardLog()

	skippedMWs := auth.ProviderMWs[func(http.Handler) http.Handler](
		flags, a, denyChecker, "seca.region", "/providers/seca.region", log)
	enforcedMWs := auth.ProviderMWs[func(http.Handler) http.Handler](
		flags, a, denyChecker, "seca.compute", "/providers/seca.compute", log)

	// Route through a real mux so r.Pattern is set for the claim extractor.
	mux := http.NewServeMux()
	mux.Handle("GET /providers/seca.region/v1/regions", wrapMWs(skippedMWs, okHandler))
	mux.Handle("GET /providers/seca.compute/v1/tenants/{tenant}/instances", wrapMWs(enforcedMWs, okHandler))

	tests := []struct {
		name       string
		path       string
		withToken  bool
		wantStatus int
	}{
		{name: "skipped provider: denying checker cannot block", path: "/providers/seca.region/v1/regions", withToken: true, wantStatus: http.StatusOK},
		{name: "skipped provider: authn still enforced", path: "/providers/seca.region/v1/regions", withToken: false, wantStatus: http.StatusUnauthorized},
		{name: "unlisted provider: authz still enforced", path: "/providers/seca.compute/v1/tenants/t1/instances", withToken: true, wantStatus: http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.withToken {
				req.Header.Set("Authorization", "Bearer "+bearerToken("alice", "s3cr3t", nil))
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d — body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

// TestRegisterFlags_AuthzSkipProvidersDefault pins the default skip list: the region
// catalog is tenant-less by spec, so seca.region must skip authorization out of the box.
func TestRegisterFlags_AuthzSkipProvidersDefault(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	var flags auth.Flags
	auth.RegisterFlags(cmd, &flags)
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	if len(flags.AuthzSkipProviders) != 1 || flags.AuthzSkipProviders[0] != "seca.region" {
		t.Errorf("AuthzSkipProviders default = %v, want [seca.region]", flags.AuthzSkipProviders)
	}
}

// TestIntegration_DownScopeFromToken verifies that the token's optional scope is
// decoded and propagated into the AuthorizationClaim as the down-scope — covering the
// contract between authn and authz. Roles are never propagated from the token.
func TestIntegration_DownScopeFromToken(t *testing.T) {
	t.Parallel()

	a := gatewayauthn.NewDummyAuthenticator(map[string]string{"bob": "p@ss"})

	var capturedClaim authzport.AuthorizationClaim
	capturing := checkerFunc(func(_ context.Context, claim authzport.AuthorizationClaim) (authzport.Decision, error) {
		capturedClaim = claim
		return authzport.DecisionAllowed, nil
	})

	log := discardLog()
	authnMW := middleware.NewAuthentication(a, log)
	authzMW := middleware.NewAuthorization(capturing, fixedExtractor, log)
	h := authnMW(authzMW(okHandler))

	scope := &resource.TokenScope{Tenants: []string{"t1"}, Regions: []string{"r1"}}
	req := httptest.NewRequest(http.MethodGet, "/instances", nil)
	req.Header.Set("Authorization", "Bearer "+bearerToken("bob", "p@ss", scope))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if capturedClaim.Subject != "bob" {
		t.Errorf("claim.Subject = %q, want %q — subject must be propagated from the bearer token", capturedClaim.Subject, "bob")
	}
	if len(capturedClaim.TokenScope.Tenants) != 1 || capturedClaim.TokenScope.Tenants[0] != "t1" {
		t.Errorf("claim.TokenScope.Tenants = %v, want [t1]", capturedClaim.TokenScope.Tenants)
	}
	if len(capturedClaim.TokenScope.Regions) != 1 || capturedClaim.TokenScope.Regions[0] != "r1" {
		t.Errorf("claim.TokenScope.Regions = %v, want [r1]", capturedClaim.TokenScope.Regions)
	}
}

// TestIntegration_DownScope_ReducesAccess drives the real seca.Evaluate end-to-end:
// bob holds an all-access admin assignment, but a token that down-scopes to a different
// tenant is denied, while a token scoped to the request's tenant is allowed. This proves
// the token cap can only narrow — never grant — the permissions RBAC would otherwise give.
func TestIntegration_DownScope_ReducesAccess(t *testing.T) {
	t.Parallel()

	adminRole := &roledom.Role{
		GlobalTenantMetadata: commondom.GlobalTenantMetadata{
			CommonMetadata: commondom.CommonMetadata{Name: "admin"},
		},
		Spec: roledom.RoleSpec{Permissions: []roledom.Permission{
			{Provider: "seca.compute", Resources: []string{"*"}, Verb: []string{"*"}},
		}},
	}
	adminAssignment := &radom.RoleAssignment{
		Spec: radom.RoleAssignmentSpec{
			Subs:   []string{"bob"},
			Roles:  []string{"admin"},
			Scopes: []radom.RoleAssignmentScope{{}}, // all tenants/regions/workspaces
		},
	}
	checker := rbacChecker(
		map[string]*roledom.Role{"admin": adminRole},
		[]*radom.RoleAssignment{adminAssignment},
	)

	a := gatewayauthn.NewDummyAuthenticator(map[string]string{"bob": "p@ss"})
	h := buildChain(a, checker) // fixedExtractor targets tenant "t1"

	tests := []struct {
		name       string
		scope      *resource.TokenScope
		wantStatus int
	}{
		{name: "no down-scope → admin allowed", scope: nil, wantStatus: http.StatusOK},
		{name: "down-scope to request tenant → allowed", scope: &resource.TokenScope{Tenants: []string{"t1"}}, wantStatus: http.StatusOK},
		{name: "down-scope to other tenant → denied", scope: &resource.TokenScope{Tenants: []string{"other"}}, wantStatus: http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/instances", nil)
			req.Header.Set("Authorization", "Bearer "+bearerToken("bob", "p@ss", tc.scope))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}
