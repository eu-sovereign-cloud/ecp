package authn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
)

func TestDummyAuthenticator(t *testing.T) {
	t.Parallel()
	users := map[string]string{
		"alice": "s3cr3t",
		"bob":   "p@ssw0rd",
	}
	a := NewDummyAuthenticator(users)

	// Helper to build a valid token from a payload.
	makeToken := func(username, password string, roles []string) string {
		p, err := json.Marshal(tokenPayload{Username: username, Password: password, Roles: roles})
		if err != nil {
			t.Fatalf("marshal token payload: %v", err)
		}
		return base64.StdEncoding.EncodeToString(p)
	}

	tests := []struct {
		name        string
		token       string
		wantSubject string
		wantRoles   []string
		wantErr     bool
	}{
		{
			name:        "valid credentials with roles",
			token:       makeToken("alice", "s3cr3t", []string{"admin", "viewer"}),
			wantSubject: "alice",
			wantRoles:   []string{"admin", "viewer"},
		},
		{
			name:        "valid credentials with empty roles",
			token:       makeToken("bob", "p@ssw0rd", []string{}),
			wantSubject: "bob",
			wantRoles:   []string{},
		},
		{
			name:        "valid credentials with nil roles normalized to empty",
			token:       makeToken("alice", "s3cr3t", nil),
			wantSubject: "alice",
			wantRoles:   []string{},
		},
		{
			name:    "wrong password",
			token:   makeToken("alice", "wrongpassword", nil),
			wantErr: true,
		},
		{
			name:    "unknown user",
			token:   makeToken("charlie", "anything", nil),
			wantErr: true,
		},
		{
			name:    "not base64",
			token:   "this is not base64!!!",
			wantErr: true,
		},
		{
			name:    "valid base64 but not JSON",
			token:   base64.StdEncoding.EncodeToString([]byte("hello world")),
			wantErr: true,
		},
		{
			name:    "missing username",
			token:   base64.StdEncoding.EncodeToString([]byte(`{"password":"s3cr3t","roles":[]}`)),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			id, err := a.Authenticate(context.Background(), tc.token)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !isUnauthorized(err) {
					t.Errorf("expected ErrUnauthorized, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id.Subject != tc.wantSubject {
				t.Errorf("subject = %q, want %q", id.Subject, tc.wantSubject)
			}
			if len(id.Roles) != len(tc.wantRoles) {
				t.Errorf("roles = %v, want %v", id.Roles, tc.wantRoles)
			}
		})
	}
}

// isUnauthorized reports whether err wraps kernel.ErrUnauthorized.
func isUnauthorized(err error) bool {
	return kernel.AsError(err) != nil && kernel.AsError(err).Kind == kernel.KindUnauthorized
}
