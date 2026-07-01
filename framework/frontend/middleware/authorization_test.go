package middleware

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	authnport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authn"
	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
)

// fakeChecker is a simple in-test Checker.
type fakeChecker struct {
	decision authzport.Decision
	err      error
}

func (f *fakeChecker) Authorize(_ context.Context, _ authzport.AuthorizationClaim) (authzport.Decision, error) {
	return f.decision, f.err
}

// errExtractor always returns an error (simulates a technical claim-extraction failure).
var errExtractor authzport.ClaimExtractor = func(_ *http.Request) (authzport.AuthorizationClaim, error) {
	return authzport.AuthorizationClaim{}, errors.New("unexpected request pattern")
}

// fixedExtractor always returns the same claim (used for extractor-happy-path tests).
func fixedExtractor(claim authzport.AuthorizationClaim) authzport.ClaimExtractor {
	return func(_ *http.Request) (authzport.AuthorizationClaim, error) { return claim, nil }
}

func TestNewAuthorization(t *testing.T) {
	t.Parallel()

	alice := &authnport.Identity{Subject: "alice", Roles: []string{"viewer"}}
	claim := authzport.AuthorizationClaim{Provider: "seca.compute", Resource: "instances", Verb: "get"}
	okExtract := fixedExtractor(claim)

	tests := []struct {
		name       string
		identity   *authnport.Identity
		checker    authzport.Checker
		extractor  authzport.ClaimExtractor
		wantStatus int
	}{
		{
			name:       "allowed: checker returns DecisionAllowed → 200",
			identity:   alice,
			checker:    &fakeChecker{decision: authzport.DecisionAllowed},
			extractor:  okExtract,
			wantStatus: http.StatusOK,
		},
		{
			name:       "denied: checker returns DecisionDenied → 403",
			identity:   alice,
			checker:    &fakeChecker{decision: authzport.DecisionDenied, err: kernel.ErrForbidden},
			extractor:  okExtract,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "technical error: checker returns DecisionError → 500",
			identity:   alice,
			checker:    &fakeChecker{decision: authzport.DecisionError, err: kernel.ErrInternal},
			extractor:  okExtract,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "extractor error → 500 (technical fault, not a policy denial)",
			identity:   alice,
			checker:    &fakeChecker{decision: authzport.DecisionAllowed},
			extractor:  errExtractor,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "missing identity (authn not run) → 401",
			// identity not injected into context
			identity:   nil,
			checker:    &fakeChecker{decision: authzport.DecisionAllowed},
			extractor:  okExtract,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mw := NewAuthorization(tc.checker, tc.extractor, slog.New(slog.NewTextHandler(io.Discard, nil)))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			if tc.identity != nil {
				r = r.WithContext(contextWithIdentity(r.Context(), tc.identity))
			}

			mw(okHandler).ServeHTTP(w, r)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestNewAuthorization_RolesFromIdentity(t *testing.T) {
	t.Parallel()
	// Verify that the authorization middleware copies the identity's roles into the claim.
	wantRoles := []string{"admin", "viewer"}
	alice := &authnport.Identity{Subject: "alice", Roles: wantRoles}

	var gotClaim authzport.AuthorizationClaim
	checker := authzport.Checker(checkerFunc(func(_ context.Context, c authzport.AuthorizationClaim) (authzport.Decision, error) {
		gotClaim = c
		return authzport.DecisionAllowed, nil
	}))
	mw := NewAuthorization(checker, fixedExtractor(authzport.AuthorizationClaim{}), discardLog)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(contextWithIdentity(r.Context(), alice))
	mw(okHandler).ServeHTTP(w, r)

	if len(gotClaim.Roles) != len(wantRoles) {
		t.Errorf("roles = %v, want %v", gotClaim.Roles, wantRoles)
	}
}

// checkerFunc adapts a function to authzport.Checker.
type checkerFunc func(context.Context, authzport.AuthorizationClaim) (authzport.Decision, error)

func (f checkerFunc) Authorize(ctx context.Context, c authzport.AuthorizationClaim) (authzport.Decision, error) {
	return f(ctx, c)
}
