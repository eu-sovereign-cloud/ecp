package middleware

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	authnport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authn"
)

// fakeAuthenticator is a simple in-test implementation of authnport.Authenticator.
type fakeAuthenticator struct {
	// valid is the token that will be accepted.
	valid string
	// identity is returned on success.
	identity *authnport.Identity
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, token string) (*authnport.Identity, error) {
	if token != f.valid {
		return nil, kernel.ErrUnauthorized
	}
	return f.identity, nil
}

// okHandler is a 200 sentinel used to verify that the next handler was reached.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})

var discardLog = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestNewAuthentication(t *testing.T) {
	t.Parallel()
	const validToken = "valid-token"
	authn := &fakeAuthenticator{
		valid:    validToken,
		identity: &authnport.Identity{Subject: "alice", Roles: []string{"admin"}},
	}
	mw := NewAuthentication(authn, discardLog)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantInCtx  bool
	}{
		{
			name:       "valid bearer token passes and injects identity",
			authHeader: "Bearer " + validToken,
			wantStatus: http.StatusOK,
			wantInCtx:  true,
		},
		{
			name:       "missing Authorization header returns 401",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantInCtx:  false,
		},
		{
			name:       "wrong prefix returns 401",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:password")),
			wantStatus: http.StatusUnauthorized,
			wantInCtx:  false,
		},
		{
			name:       "invalid token returns 401",
			authHeader: "Bearer wrong-token",
			wantStatus: http.StatusUnauthorized,
			wantInCtx:  false,
		},
		{
			name:       "bearer with empty token returns 401",
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
			wantInCtx:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var capturedCtx context.Context
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedCtx = r.Context()
				w.WriteHeader(http.StatusOK)
			})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				r.Header.Set("Authorization", tc.authHeader)
			}

			mw(next).ServeHTTP(w, r)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
			if tc.wantInCtx {
				id, ok := IdentityFromContext(capturedCtx)
				if !ok || id == nil {
					t.Errorf("expected identity in context, got none")
				} else if id.Subject != "alice" {
					t.Errorf("subject = %q, want %q", id.Subject, "alice")
				}
			}
		})
	}
}

// errAuthenticator is an in-test Authenticator that always returns a fixed error.
// Used to simulate authenticator failures (credential or technical) without I/O.
type errAuthenticator struct {
	err error
}

func (e *errAuthenticator) Authenticate(_ context.Context, _ string) (*authnport.Identity, error) {
	return nil, e.err
}

// TestNewAuthentication_TechnicalError verifies that an ErrInternal returned by the
// authenticator yields HTTP 500, not 401 — confirming that infrastructure failures are
// never silently disguised as authentication failures.
func TestNewAuthentication_TechnicalError(t *testing.T) {
	t.Parallel()
	mw := NewAuthentication(&errAuthenticator{err: kernel.ErrInternal}, discardLog)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer some-token")
	mw(okHandler).ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d — technical auth failures must not be disguised as 401", w.Code)
	}
}

func TestIdentityFromContext_MissingReturnsNil(t *testing.T) {
	t.Parallel()
	id, ok := IdentityFromContext(context.Background())
	if ok || id != nil {
		t.Errorf("expected (nil, false) from empty context, got (%v, %v)", id, ok)
	}
}
