//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	authhelper "github.com/eu-sovereign-cloud/ecp/test/internal/authhelper"

	resource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
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
	if !authhelper.AuthEnabled() {
		t.Skip("E2E_AUTH_ENABLED=false: skipping authz tests")
	}

	baseURL := fmt.Sprintf("http://localhost:%d", globalLocalPort)

	t.Run("regions are authn-only (seca.region skips authorization)", func(t *testing.T) {
		// The region catalog is tenant-less by spec, so tenant-scoped RBAC cannot govern
		// it: the gateway serves seca.region without the authorization middleware
		// (--authz-skip-providers, default "seca.region"). "nobody" has valid credentials
		// but no RoleAssignment at all, yet listing regions succeeds — authentication
		// alone is required (the 401 cases are covered by TestAuthn).
		editor := authhelper.IdentityEditor("nobody", "nobody-pass")
		client, err := regionv1.NewClientWithResponses(baseURL+"/providers/seca.region", regionv1.WithRequestEditorFn(editor))
		if err != nil {
			t.Fatalf("create client: %v", err)
		}
		resp, err := client.ListRegionsWithResponse(context.Background(), &regionv1.ListRegionsParams{})
		if err != nil {
			t.Fatalf("list regions: %v", err)
		}
		if resp.StatusCode() != http.StatusOK {
			t.Errorf("nobody list regions (authn-only): want 200, got %d", resp.StatusCode())
		}
	})

	t.Run("alice cannot create a role (provider mismatch: seca.authorization)", func(t *testing.T) {
		// alice's assignment grants only e2e-region-viewer (seca.region), not seca.authorization.
		editor := authhelper.IdentityEditor("alice", "alice-pass")
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

	t.Run("nobody gets 403 (valid creds, no RoleAssignment grants seca.authorization)", func(t *testing.T) {
		// "nobody" exists in users-configmap.yaml but has no RoleAssignment, so every
		// RBAC-governed provider denies them: no grant for seca.authorization → 403.
		editor := authhelper.IdentityEditor("nobody", "nobody-pass")
		client, err := authv1.NewClientWithResponses(baseURL+"/providers/seca.authorization", authv1.WithRequestEditorFn(editor))
		if err != nil {
			t.Fatalf("create client: %v", err)
		}
		resp, err := client.ListRolesWithResponse(context.Background(), testTenant, &authv1.ListRolesParams{})
		if err != nil {
			t.Fatalf("list roles: %v", err)
		}
		if resp.StatusCode() != http.StatusForbidden {
			t.Errorf("nobody list roles: want 403, got %d", resp.StatusCode())
		}
	})

	t.Run("erin is denied admin ops in test-tenant (ra-wrong-tenant scoped to other-tenant)", func(t *testing.T) {
		// erin has ra-wrong-tenant granting e2e-admin, but scoped to Tenants=["other-tenant"],
		// so test-tenant is out of scope and no other assignment covers her → 403.
		editor := authhelper.IdentityEditor("erin", "erin-pass")
		client, err := authv1.NewClientWithResponses(baseURL+"/providers/seca.authorization", authv1.WithRequestEditorFn(editor))
		if err != nil {
			t.Fatalf("create client: %v", err)
		}
		resp, err := client.ListRolesWithResponse(context.Background(), testTenant, &authv1.ListRolesParams{})
		if err != nil {
			t.Fatalf("list roles: %v", err)
		}
		if resp.StatusCode() != http.StatusForbidden {
			t.Errorf("erin list roles in test-tenant: want 403, got %d", resp.StatusCode())
		}
	})

	t.Run("admin down-scoped to other-tenant is denied in test-tenant (tenant cap)", func(t *testing.T) {
		// admin (ra-admin) has all-access, but a token down-scoped to other-tenant must not
		// authorize operations in test-tenant; a token scoped to test-tenant still works.
		denied := authhelper.ScopedEditor(authhelper.DefaultAuthUser, authhelper.DefaultAuthPassword, &resource.TokenScope{Tenants: []string{"other-tenant"}})
		client, err := authv1.NewClientWithResponses(baseURL+"/providers/seca.authorization", authv1.WithRequestEditorFn(denied))
		if err != nil {
			t.Fatalf("create client: %v", err)
		}
		resp, err := client.ListRolesWithResponse(context.Background(), testTenant, &authv1.ListRolesParams{})
		if err != nil {
			t.Fatalf("list roles (down-scoped): %v", err)
		}
		if resp.StatusCode() != http.StatusForbidden {
			t.Errorf("admin down-scoped to other-tenant: want 403 in test-tenant, got %d", resp.StatusCode())
		}

		allowed := authhelper.ScopedEditor(authhelper.DefaultAuthUser, authhelper.DefaultAuthPassword, &resource.TokenScope{Tenants: []string{testTenant}})
		client2, err := authv1.NewClientWithResponses(baseURL+"/providers/seca.authorization", authv1.WithRequestEditorFn(allowed))
		if err != nil {
			t.Fatalf("create client: %v", err)
		}
		resp2, err := client2.ListRolesWithResponse(context.Background(), testTenant, &authv1.ListRolesParams{})
		if err != nil {
			t.Fatalf("list roles (in-scope): %v", err)
		}
		if resp2.StatusCode() != http.StatusOK {
			t.Errorf("admin down-scoped to test-tenant: want 200, got %d", resp2.StatusCode())
		}
	})
}
