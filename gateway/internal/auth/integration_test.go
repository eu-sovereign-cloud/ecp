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

	"github.com/eu-sovereign-cloud/ecp/framework/frontend/middleware"
	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	authnport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authn"
	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
	gatewayauthn "github.com/eu-sovereign-cloud/ecp/gateway/internal/authn"
)

// ── test doubles ─────────────────────────────────────────────────────────────

// checkerFunc is a test-only implementation of authzport.Checker backed by a
// function literal.
type checkerFunc func(ctx context.Context, claim authzport.AuthorizationClaim) error

func (f checkerFunc) Authorize(ctx context.Context, claim authzport.AuthorizationClaim) error {
	return f(ctx, claim)
}

// allowChecker always permits.
var allowChecker checkerFunc = func(_ context.Context, _ authzport.AuthorizationClaim) error {
	return nil
}

// denyChecker always forbids.
var denyChecker checkerFunc = func(_ context.Context, _ authzport.AuthorizationClaim) error {
	return kernel.ErrForbidden
}

// ── helpers ───────────────────────────────────────────────────────────────────

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// bearerToken encodes a Dummy-auth bearer token from the given credentials.
func bearerToken(username, password string, roles []string) string {
	type payload struct {
		Username string   `json:"username"`
		Password string   `json:"password"`
		Roles    []string `json:"roles"`
	}
	b, err := json.Marshal(payload{Username: username, Password: password, Roles: roles})
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
	req.Header.Set("Authorization", "Bearer "+bearerToken("alice", "s3cr3t", []string{"viewer"}))
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
	req.Header.Set("Authorization", "Bearer "+bearerToken("alice", "wrongpass", []string{}))
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
	req.Header.Set("Authorization", "Bearer "+bearerToken("alice", "s3cr3t", []string{"admin"}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", w.Code)
	}
}

// TestIntegration_RolesFromToken verifies that the identity's roles (decoded from
// the bearer token) are propagated into the AuthorizationClaim by the authorization
// middleware — covering the key contract between authn and authz.
func TestIntegration_RolesFromToken(t *testing.T) {
	t.Parallel()

	a := gatewayauthn.NewDummyAuthenticator(map[string]string{"bob": "p@ss"})

	var capturedClaim authzport.AuthorizationClaim
	capturing := checkerFunc(func(_ context.Context, claim authzport.AuthorizationClaim) error {
		capturedClaim = claim
		return nil
	})

	log := discardLog()
	authnMW := middleware.NewAuthentication(a, log)
	authzMW := middleware.NewAuthorization(capturing, fixedExtractor, log)
	h := authnMW(authzMW(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/instances", nil)
	req.Header.Set("Authorization", "Bearer "+bearerToken("bob", "p@ss", []string{"admin", "viewer"}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if len(capturedClaim.Roles) != 2 {
		t.Fatalf("want 2 roles, got %d: %v", len(capturedClaim.Roles), capturedClaim.Roles)
	}
	if capturedClaim.Roles[0] != "admin" || capturedClaim.Roles[1] != "viewer" {
		t.Errorf("claim.Roles = %v, want [admin viewer]", capturedClaim.Roles)
	}
}
