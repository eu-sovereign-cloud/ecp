//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	authv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// TestAuthz exercises SECA RBAC authorization on the global gateway.
// These tests rely on the Role and RoleAssignment fixtures from
// test/e2e/deploy/test-data/roles.yaml and role-assignments.yaml, and on the
// Dummy authenticator being enabled with the users from users-configmap.yaml.
// Skipped when E2E_AUTH_ENABLED=false.
func TestAuthz(t *testing.T) {
	if !authEnabled() {
		t.Skip("E2E_AUTH_ENABLED=false: skipping authz tests")
	}

	baseURL := fmt.Sprintf("http://localhost:%d", globalLocalPort)

	t.Run("ra-wildcard: any caller with region-viewer role can list regions", func(t *testing.T) {
		// ra-wildcard has Subs=["*"] and Roles=["e2e-region-viewer"] in scope all.
		// erin also has ra-wrong-tenant (scope other-tenant) with e2e-admin, which does
		// not help here. The request hits /v1/regions (resource "v1/regions") with no
		// tenant — ra-wildcard's empty scope covers it; e2e-region-viewer resource
		// pattern "v1/regions" matches.
		editor := identityEditor("erin", "erin-pass", []string{"e2e-region-viewer"})
		client, err := regionv1.NewClientWithResponses(baseURL+"/providers/seca.region", regionv1.WithRequestEditorFn(editor))
		if err != nil {
			t.Fatalf("create client: %v", err)
		}
		resp, err := client.ListRegionsWithResponse(context.Background(), &regionv1.ListRegionsParams{})
		if err != nil {
			t.Fatalf("list regions: %v", err)
		}
		if resp.StatusCode() != http.StatusOK {
			t.Errorf("erin via ra-wildcard: want 200, got %d", resp.StatusCode())
		}
	})

	t.Run("alice cannot create a role (provider mismatch: seca.authorization)", func(t *testing.T) {
		// alice only has e2e-region-viewer which covers seca.region, not seca.authorization.
		editor := identityEditor("alice", "alice-pass", []string{"e2e-region-viewer"})
		client, err := authv1.NewClientWithResponses(baseURL+"/providers/seca.authorization", authv1.WithRequestEditorFn(editor))
		if err != nil {
			t.Fatalf("create client: %v", err)
		}
		resp, err := client.CreateOrUpdateRoleWithResponse(
			context.Background(),
			testTenant,
			"e2e-alice-forbidden-role",
			&authv1.CreateOrUpdateRoleParams{},
			schema.Role{Spec: schema.RoleSpec{Permissions: []schema.Permission{{Provider: "seca.region", Resources: []string{"regions"}, Verb: []string{"get"}}}}},
		)
		if err != nil {
			t.Fatalf("create role: %v", err)
		}
		if resp.StatusCode() != http.StatusForbidden {
			t.Errorf("alice create role: want 403, got %d", resp.StatusCode())
		}
	})

	t.Run("nobody gets 403 (valid creds, no RoleAssignment)", func(t *testing.T) {
		// "nobody" exists in users-configmap.yaml but has no RoleAssignment.
		// ra-wildcard gives e2e-region-viewer to everyone, but nobody's token carries
		// e2e-admin (not e2e-region-viewer), so the intersection is empty → 403.
		editor := identityEditor("nobody", "nobody-pass", []string{"e2e-admin"})
		client, err := regionv1.NewClientWithResponses(baseURL+"/providers/seca.region", regionv1.WithRequestEditorFn(editor))
		if err != nil {
			t.Fatalf("create client: %v", err)
		}
		resp, err := client.ListRegionsWithResponse(context.Background(), &regionv1.ListRegionsParams{})
		if err != nil {
			t.Fatalf("list regions: %v", err)
		}
		if resp.StatusCode() != http.StatusForbidden {
			t.Errorf("nobody list regions: want 403, got %d", resp.StatusCode())
		}
	})

	t.Run("erin is denied admin ops in test-tenant (ra-wrong-tenant scoped to other-tenant)", func(t *testing.T) {
		// erin has ra-wrong-tenant whose scope is Tenants=["other-tenant"].
		// Even with e2e-admin role in the token, test-tenant is out of scope.
		// ra-wildcard gives e2e-region-viewer but erin's token carries e2e-admin → no intersection.
		editor := identityEditor("erin", "erin-pass", []string{"e2e-admin"})
		client, err := authv1.NewClientWithResponses(baseURL+"/providers/seca.authorization", authv1.WithRequestEditorFn(editor))
		if err != nil {
			t.Fatalf("create client: %v", err)
		}
		resp, err := client.ListRolesWithResponse(context.Background(), testTenant, &authv1.ListRolesParams{})
		if err != nil {
			t.Fatalf("list roles: %v", err)
		}
		// ra-wildcard gives e2e-region-viewer but not e2e-admin; ra-wrong-tenant
		// gives e2e-admin but scope excludes test-tenant → net result: 403.
		if resp.StatusCode() != http.StatusForbidden {
			t.Errorf("erin list roles in test-tenant: want 403, got %d", resp.StatusCode())
		}
	})
}
