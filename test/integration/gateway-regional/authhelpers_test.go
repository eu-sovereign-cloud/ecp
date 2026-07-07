//go:build integration

package integration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"

	resource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

// defaultAuthUser is the username used by the default SDK clients.
// It must match a Subs entry in ra-admin (role-assignments.yaml).
const defaultAuthUser = "admin"

// defaultAuthPassword is the password for defaultAuthUser (matches users-configmap.yaml).
const defaultAuthPassword = "e2e-admin-pass"

// authEnabled reports whether auth is expected to be active in the deployed gateway.
// Set E2E_AUTH_ENABLED=false to skip auth-specific assertions and use unauthenticated
// clients (for running the suite against a gateway deployed without --auth-enabled).
func authEnabled() bool {
	return os.Getenv("E2E_AUTH_ENABLED") != "false"
}

// makeBearerToken encodes a Dummy authenticator bearer token.
// The token is base64(JSON{"username":…,"password":…,"scope":…}); the optional scope reuses
// [resource.TokenScope] and its json tags. Roles are never carried by the token — they are
// resolved from RoleAssignments in the caller's tenant namespace.
func makeBearerToken(username, password string, scope *resource.TokenScope) string {
	type payload struct {
		Username string               `json:"username"`
		Password string               `json:"password"`
		Scope    *resource.TokenScope `json:"scope,omitempty"`
	}
	b, err := json.Marshal(payload{Username: username, Password: password, Scope: scope})
	if err != nil {
		panic("makeBearerToken: marshal failed: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(b)
}

// bearerEditor returns a request editor that injects "Authorization: Bearer <token>".
func bearerEditor(token string) func(ctx context.Context, req *http.Request) error {
	return func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}

// adminEditor returns the request editor for the default admin identity.
// When E2E_AUTH_ENABLED=false it returns a no-op editor so clients work unchanged.
func adminEditor() func(ctx context.Context, req *http.Request) error {
	if !authEnabled() {
		return func(_ context.Context, _ *http.Request) error { return nil }
	}
	return bearerEditor(makeBearerToken(defaultAuthUser, defaultAuthPassword, nil))
}

// identityEditor returns a request editor for the given username/password.
func identityEditor(username, password string) func(ctx context.Context, req *http.Request) error {
	return bearerEditor(makeBearerToken(username, password, nil))
}

// scopedEditor is like identityEditor but attaches a token down-scope: tenant/region/workspace
// caps that can only narrow the caller's permissions, never grant new ones.
func scopedEditor(username, password string, scope *resource.TokenScope) func(ctx context.Context, req *http.Request) error {
	return bearerEditor(makeBearerToken(username, password, scope))
}
