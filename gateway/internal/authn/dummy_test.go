package authn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

func TestDummyAuthenticator(t *testing.T) {
	t.Parallel()
	users := map[string]string{
		"alice": "s3cr3t",
		"bob":   "p@ssw0rd",
	}
	a := NewDummyAuthenticator(users)

	// Helper to build a valid token from a payload.
	makeToken := func(username, password string, scope *resource.TokenScope) string {
		p, err := json.Marshal(tokenPayload{Username: username, Password: password, Scope: scope})
		if err != nil {
			t.Fatalf("marshal token payload: %v", err)
		}
		return base64.StdEncoding.EncodeToString(p)
	}

	tests := []struct {
		name        string
		token       string
		wantSubject string
		wantScope   resource.TokenScope
		wantTenants []string
		wantErr     bool
	}{
		{
			// The "tenants" list stands in for the membership a real issuer stamps.
			name:        "tenants claim becomes the identity's membership",
			token:       base64.StdEncoding.EncodeToString([]byte(`{"username":"alice","password":"s3cr3t","tenants":["t1","t2"]}`)),
			wantSubject: "alice",
			wantTenants: []string{"t1", "t2"},
		},
		{
			name:        "valid credentials without scope",
			token:       makeToken("alice", "s3cr3t", nil),
			wantSubject: "alice",
		},
		{
			name: "valid credentials with down-scope",
			token: makeToken("bob", "p@ssw0rd", &resource.TokenScope{
				Tenants:    []string{"t1"},
				Regions:    []string{"r1"},
				Workspaces: []string{"w1"},
			}),
			wantSubject: "bob",
			wantScope: resource.TokenScope{
				Tenants:    []string{"t1"},
				Regions:    []string{"r1"},
				Workspaces: []string{"w1"},
			},
		},
		{
			// Roles are never read from the token; a stray "roles" field must be ignored.
			name:        "roles field in token is ignored",
			token:       base64.StdEncoding.EncodeToString([]byte(`{"username":"alice","password":"s3cr3t","roles":["admin"]}`)),
			wantSubject: "alice",
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
			token:   base64.StdEncoding.EncodeToString([]byte(`{"password":"s3cr3t"}`)),
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
			if !reflect.DeepEqual(id.MemberTenants, tc.wantTenants) {
				t.Errorf("member tenants = %v, want %v", id.MemberTenants, tc.wantTenants)
			}
			if !reflect.DeepEqual(id.TokenScope, tc.wantScope) {
				t.Errorf("token scope = %+v, want %+v", id.TokenScope, tc.wantScope)
			}
		})
	}
}

// isUnauthorized reports whether err wraps kernel.ErrUnauthorized.
func isUnauthorized(err error) bool {
	return kernel.AsError(err) != nil && kernel.AsError(err).Kind == kernel.KindUnauthorized
}
