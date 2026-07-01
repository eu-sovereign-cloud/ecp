//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
)

// TestAuthn exercises the bearer-token authentication middleware on the global gateway.
// These tests are skipped when E2E_AUTH_ENABLED=false (unauthenticated mode).
func TestAuthn(t *testing.T) {
	if !authEnabled() {
		t.Skip("E2E_AUTH_ENABLED=false: skipping authn tests")
	}

	listRegionsURL := fmt.Sprintf("http://localhost:%d/providers/seca.region/v1/regions", globalLocalPort)

	t.Run("valid token returns 200", func(t *testing.T) {
		token := makeBearerToken(defaultAuthUser, defaultAuthPassword, defaultAuthRoles)
		req, _ := http.NewRequest(http.MethodGet, listRegionsURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("want 200 for valid token, got %d", resp.StatusCode)
		}
	})

	t.Run("missing Authorization header returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, listRegionsURL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("want 401 for missing token, got %d", resp.StatusCode)
		}
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		token := makeBearerToken(defaultAuthUser, "wrong-password", defaultAuthRoles)
		req, _ := http.NewRequest(http.MethodGet, listRegionsURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("want 401 for wrong password, got %d", resp.StatusCode)
		}
	})

	t.Run("malformed base64 token returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, listRegionsURL, nil)
		req.Header.Set("Authorization", "Bearer not!valid!base64!!!")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("want 401 for malformed token, got %d", resp.StatusCode)
		}
	})
}
