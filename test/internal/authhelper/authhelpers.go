//go:build authhelper

package authhelper

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	resource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

// DefaultAuthUser is the username used by the default SDK clients.
// It must match a Subs entry in ra-admin (role-assignments.yaml).
const DefaultAuthUser = "admin"

// DefaultAuthPassword is the password for DefaultAuthUser (matches users-configmap.yaml).
const DefaultAuthPassword = "e2e-admin-pass"

// AuthEnabled reports whether auth is expected to be active in the deployed gateway.
// Set E2E_AUTH_ENABLED=false to skip auth-specific assertions and use unauthenticated
// clients (for running the suite against a gateway deployed without --auth-enabled).
func AuthEnabled() bool {
	return os.Getenv("E2E_AUTH_ENABLED") != "false"
}

// JWTAuth reports whether the deployed gateways verify signed JWTs instead of dummy
// tokens. It reads the same AUTH_PLUGIN the Makefile substitutes into the manifests
// (default "dummy"), so the stack and the suites can never disagree on the token
// format. Plugin-specific tests use it to skip themselves.
func JWTAuth() bool {
	return os.Getenv("AUTH_PLUGIN") == "jwt"
}

// Token mints a bearer token for the deployed authenticator: a JWT with the username
// as "sub" when the gateways run the jwt plugin, a dummy token otherwise (the JWT
// plugin trusts the signature, so the password is unused). Both plugins feed the same
// Identity.Subject, so RBAC resolves identically either way.
func Token(username, password string, scope *resource.TokenScope) string {
	if JWTAuth() {
		return SignJWT(JWTKey(), username, scope, time.Now().Add(time.Hour))
	}
	return MakeBearerToken(username, password, scope)
}

// MakeBearerToken encodes a Dummy authenticator bearer token.
// The token is base64(JSON{"username":…,"password":…,"scope":…}); the optional scope reuses
// [resource.TokenScope] and its json tags. Roles are never carried by the token — they are
// resolved from RoleAssignments in the caller's tenant namespace.
func MakeBearerToken(username, password string, scope *resource.TokenScope) string {
	type payload struct {
		Username string               `json:"username"`
		Password string               `json:"password"`
		Scope    *resource.TokenScope `json:"scope,omitempty"`
	}
	b, err := json.Marshal(payload{Username: username, Password: password, Scope: scope})
	if err != nil {
		panic("MakeBearerToken: marshal failed: " + err.Error())
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

// AdminEditor returns the request editor for the default admin identity.
// When E2E_AUTH_ENABLED=false it returns a no-op editor so clients work unchanged.
func AdminEditor() func(ctx context.Context, req *http.Request) error {
	if !AuthEnabled() {
		return func(_ context.Context, _ *http.Request) error { return nil }
	}
	return bearerEditor(Token(DefaultAuthUser, DefaultAuthPassword, nil))
}

// IdentityEditor returns a request editor for the given username/password.
// Panics if called when auth is disabled (callers should guard with AuthEnabled()).
func IdentityEditor(username, password string) func(ctx context.Context, req *http.Request) error {
	return bearerEditor(Token(username, password, nil))
}

// ScopedEditor is like IdentityEditor but attaches a token down-scope: tenant/region/workspace
// caps that can only narrow the caller's permissions, never grant new ones.
func ScopedEditor(username, password string, scope *resource.TokenScope) func(ctx context.Context, req *http.Request) error {
	return bearerEditor(Token(username, password, scope))
}

// --- JWT authenticator --------------------------------------------------------
//
// Both gateways are deployed with the plugin named by AUTH_PLUGIN; the helpers
// below mint (or deliberately forge) the tokens the jwt plugin accepts.

// jwtPrivateKeyPEM is the ES256 private key the suite signs e2e JWTs with. Its
// public half is deployed in internal/deploy/gateway-*/jwt-key-secret.yaml
// and passed to the gateways via --jwt-secret.
//
// WARNING: a published test fixture, not a secret. Never use it in production.
const jwtPrivateKeyPEM = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg4HzjtpKtnZr+3LTU
D79whZ+HfRp0q9ij/bkoo7q2YUqhRANCAAQAI2LTL/6j720JbMDsfo350lmbvgwm
0TVJaWuk2T0qcpwrOcD1sRZ/3r/gdkgaE3vERf0v3EQ7GhzMo03mVTVh
-----END PRIVATE KEY-----`

// JWTSigningMethod is the algorithm the global gateway is deployed with
// (JWT_SIGNING_METHOD / --jwt-signing-method). Tokens signed with any other
// method are rejected by jwt.WithValidMethods.
var JWTSigningMethod = jwt.SigningMethodES256

// JWTKey parses the fixture signing key. Panics on failure: the key is a
// compile-time constant, so a parse error is a bug, not a test condition.
func JWTKey() *ecdsa.PrivateKey {
	key, err := jwt.ParseECPrivateKeyFromPEM([]byte(jwtPrivateKeyPEM))
	if err != nil {
		panic("JWTKey: parse fixture key failed: " + err.Error())
	}
	return key
}

// SignJWT builds a standard signed JWT for the given subject. The subject becomes
// Identity.Subject and is matched against RoleAssignment.Spec.Subs, exactly as the
// dummy token's username is; the optional scope down-scopes the caller. Pass a key
// other than JWTKey() to forge a token the gateway must reject.
func SignJWT(key *ecdsa.PrivateKey, subject string, scope *resource.TokenScope, exp time.Time) string {
	claims := jwt.MapClaims{"sub": subject, "exp": exp.Unix()}
	if scope != nil {
		claims["scope"] = scope
	}
	signed, err := jwt.NewWithClaims(JWTSigningMethod, claims).SignedString(key)
	if err != nil {
		panic("SignJWT: signing failed: " + err.Error())
	}
	return signed
}
