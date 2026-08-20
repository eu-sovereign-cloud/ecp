//go:build e2e

package e2e

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	resource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	authhelper "github.com/eu-sovereign-cloud/ecp/test/internal/authhelper"
)

// TestJWTAuthn exercises the JWT authenticator on the deployed global gateway,
// which AUTH_PLUGIN=jwt configures with --auth-plugin=jwt
// --jwt-signing-method=ES256 and the public key in internal/deploy/gateway-values.yaml. Run the
// stack with AUTH_PLUGIN=jwt to cover it; under the dummy plugin (the default)
// these cases do not apply and the suite skips them.
//
// This is the only test that covers the flag → key file → ParseVerifyKey →
// authenticator wiring: the unit tests build the authenticator from an
// already-parsed key, so a gateway that mishandles the key file still passes them.
func TestJWTAuthn(t *testing.T) {
	if !authhelper.AuthEnabled() {
		t.Skip("E2E_AUTH_ENABLED=false: skipping JWT authn tests")
	}
	if !authhelper.JWTAuth() {
		t.Skip("AUTH_PLUGIN is not jwt: the gateways verify dummy tokens")
	}

	// seca.region is authn-only (--authz-skip-providers), so its status isolates
	// authentication from any RBAC decision: a valid token is 200, full stop.
	listRegionsURL := globalURL + "/providers/seca.region/v1/regions"
	// seca.authorization is RBAC-governed, so it is where a 403 is meaningful.
	listRolesURL := globalURL + "/providers/seca.authorization/v1/tenants/" + testTenant + "/roles"

	hour := time.Now().Add(time.Hour)

	t.Run("valid JWT is accepted", func(t *testing.T) {
		token := authhelper.SignJWT(authhelper.JWTKey(), authhelper.DefaultAuthUser, nil, nil, hour)
		requireStatus(t, http.MethodGet, listRegionsURL, token, http.StatusOK)
	})

	t.Run("valid JWT passes authn and authz on an RBAC-governed provider", func(t *testing.T) {
		// admin holds ra-admin, so the subject carried by the JWT resolves to the
		// same roles a dummy token's username would: proof the plugins are
		// interchangeable from the authorization layer's point of view.
		token := authhelper.SignJWT(authhelper.JWTKey(), authhelper.DefaultAuthUser, nil, nil, hour)
		requireStatus(t, http.MethodGet, listRolesURL, token, http.StatusOK)
	})

	t.Run("missing Authorization header is rejected", func(t *testing.T) {
		requireStatus(t, http.MethodGet, listRegionsURL, "", http.StatusUnauthorized)
	})

	t.Run("JWT signed with an unknown key is rejected", func(t *testing.T) {
		forged, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		token := authhelper.SignJWT(forged, authhelper.DefaultAuthUser, nil, nil, hour)
		requireStatus(t, http.MethodGet, listRegionsURL, token, http.StatusUnauthorized)
	})

	t.Run("expired JWT is rejected", func(t *testing.T) {
		token := authhelper.SignJWT(authhelper.JWTKey(), authhelper.DefaultAuthUser, nil, nil, time.Now().Add(-time.Hour))
		requireStatus(t, http.MethodGet, listRegionsURL, token, http.StatusUnauthorized)
	})

	t.Run("JWT without an expiry is rejected", func(t *testing.T) {
		// The gateway parses with jwt.WithExpirationRequired, so a token that never
		// expires must not be honoured even though its signature is valid.
		claims := jwt.MapClaims{"sub": authhelper.DefaultAuthUser}
		token, err := jwt.NewWithClaims(authhelper.JWTSigningMethod, claims).SignedString(authhelper.JWTKey())
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}
		requireStatus(t, http.MethodGet, listRegionsURL, token, http.StatusUnauthorized)
	})

	t.Run("algorithm confusion attempt is rejected", func(t *testing.T) {
		// The classic attack: re-sign the token as HS256 using the gateway's own
		// public key as the HMAC secret. jwt.WithValidMethods pins ES256 and must
		// reject it on the header alone, before the key is ever consulted.
		der, err := x509.MarshalPKIXPublicKey(&authhelper.JWTKey().PublicKey)
		if err != nil {
			t.Fatalf("failed to marshal public key: %v", err)
		}
		pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
		claims := jwt.MapClaims{"sub": authhelper.DefaultAuthUser, "exp": hour.Unix()}
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(pubPEM)
		if err != nil {
			t.Fatalf("failed to sign HS256 token: %v", err)
		}
		requireStatus(t, http.MethodGet, listRegionsURL, token, http.StatusUnauthorized)
	})

	t.Run("dummy-format token is rejected", func(t *testing.T) {
		// The same credentials the regional gateway accepts must not open the
		// global one: proof it really switched plugins rather than accepting
		// anything that looks like a bearer token.
		token := authhelper.MakeBearerToken(authhelper.DefaultAuthUser, authhelper.DefaultAuthPassword, nil, nil)
		requireStatus(t, http.MethodGet, listRegionsURL, token, http.StatusUnauthorized)
	})

	t.Run("JWT subject drives RBAC", func(t *testing.T) {
		// "nobody" authenticates but holds no RoleAssignment: authn-only providers
		// answer, RBAC-governed ones deny. A 403 (not 401) proves the JWT's subject
		// reached the authorization layer.
		token := authhelper.SignJWT(authhelper.JWTKey(), "nobody", nil, nil, hour)
		requireStatus(t, http.MethodGet, listRegionsURL, token, http.StatusOK)
		requireStatus(t, http.MethodGet, listRolesURL, token, http.StatusForbidden)
	})

	t.Run("JWT from another issuer is rejected", func(t *testing.T) {
		// Signed with the key the gateway trusts, but minted by someone else: the
		// --jwt-issuer check is the only thing standing between the two, so a 401
		// here proves the flag reached the deployed authenticator. The claim matrix
		// itself (wrong/missing iss and aud) is covered in gateway/internal/authn.
		claims := jwt.MapClaims{
			"sub": authhelper.DefaultAuthUser, "exp": hour.Unix(),
			"iss": "https://evil.example", "aud": authhelper.JWTAudience,
		}
		token, err := jwt.NewWithClaims(authhelper.JWTSigningMethod, claims).SignedString(authhelper.JWTKey())
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}
		requireStatus(t, http.MethodGet, listRegionsURL, token, http.StatusUnauthorized)
	})

	t.Run("tenants claim gates the request tenant", func(t *testing.T) {
		// admin is unrestricted by RBAC, so the denial can only come from the
		// issuer-asserted membership travelling from the JWT into the authorization
		// claim — the same path the down-scope takes, but one the caller cannot omit.
		outsider := authhelper.MemberToken(authhelper.DefaultAuthUser, authhelper.DefaultAuthPassword, []string{"other-tenant"})
		requireStatus(t, http.MethodGet, listRolesURL, outsider, http.StatusForbidden)

		member := authhelper.MemberToken(authhelper.DefaultAuthUser, authhelper.DefaultAuthPassword, []string{testTenant})
		requireStatus(t, http.MethodGet, listRolesURL, member, http.StatusOK)
		// The gate is authorization, not authentication: an authn-only provider still answers.
		requireStatus(t, http.MethodGet, listRegionsURL, outsider, http.StatusOK)
	})

	t.Run("JWT scope claim down-scopes the caller", func(t *testing.T) {
		// admin is unrestricted by RBAC, so any denial here comes from the token's
		// own scope cap travelling from the JWT into the authorization claim.
		capped := &resource.TokenScope{Tenants: []string{"other-tenant"}}
		token := authhelper.SignJWT(authhelper.JWTKey(), authhelper.DefaultAuthUser, capped, nil, hour)
		requireStatus(t, http.MethodGet, listRolesURL, token, http.StatusForbidden)

		inScope := &resource.TokenScope{Tenants: []string{testTenant}}
		token = authhelper.SignJWT(authhelper.JWTKey(), authhelper.DefaultAuthUser, inScope, nil, hour)
		requireStatus(t, http.MethodGet, listRolesURL, token, http.StatusOK)
	})
}

// requireStatus issues the request with the given bearer token (omitted when
// token is empty) and fails the test unless the response status matches want.
func requireStatus(t *testing.T, method, url, token string, want int) {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request to %s failed: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		t.Errorf("GET %s: got %d, want %d", url, resp.StatusCode, want)
	}
}
