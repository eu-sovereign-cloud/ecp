//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	resource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	storagev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	workspacev1sdk "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// TestRegionalAuthz exercises SECA RBAC authorization on the regional gateway.
// These tests rely on the Role and RoleAssignment fixtures from
// test/e2e/deploy/test-data/roles.yaml and role-assignments.yaml, and on the
// Dummy authenticator being enabled with the users from users-configmap.yaml.
// Skipped when E2E_AUTH_ENABLED=false.
func TestRegionalAuthz(t *testing.T) {
	if !authEnabled() {
		t.Skip("E2E_AUTH_ENABLED=false: skipping regional authz tests")
	}

	regionalBaseURL := fmt.Sprintf("http://localhost:%d", regionalLocalPort)

	t.Run("bob can list block-storages (e2e-storage-viewer scoped to itbg-bergamo)", func(t *testing.T) {
		// bob has ra-bob-scoped: seca.storage viewer scoped to itbg-bergamo region.
		// The regional gateway runs in itbg-bergamo so the scope matches.
		editor := identityEditor("bob", "bob-pass")
		client, err := storagev1.NewClientWithResponses(regionalBaseURL+"/providers/seca.storage", storagev1.WithRequestEditorFn(editor))
		if err != nil {
			t.Fatalf("create storage client: %v", err)
		}
		resp, err := client.ListBlockStoragesWithResponse(context.Background(), testTenant, testWorkspace, &storagev1.ListBlockStoragesParams{})
		if err != nil {
			t.Fatalf("list block storages: %v", err)
		}
		if resp.StatusCode() != http.StatusOK {
			t.Errorf("bob list storage: want 200, got %d", resp.StatusCode())
		}
	})

	t.Run("bob down-scoped to another region is denied (region cap)", func(t *testing.T) {
		// bob's RBAC (ra-bob-scoped) allows storage in itbg-bergamo, which is the region
		// this gateway serves. A token down-scoped to a different region must not authorize
		// the request even though RBAC would; a token scoped to itbg-bergamo still works.
		denied := scopedEditor("bob", "bob-pass", &resource.TokenScope{Regions: []string{"other-region"}})
		client, err := storagev1.NewClientWithResponses(regionalBaseURL+"/providers/seca.storage", storagev1.WithRequestEditorFn(denied))
		if err != nil {
			t.Fatalf("create storage client: %v", err)
		}
		resp, err := client.ListBlockStoragesWithResponse(context.Background(), testTenant, testWorkspace, &storagev1.ListBlockStoragesParams{})
		if err != nil {
			t.Fatalf("list block storages (down-scoped): %v", err)
		}
		if resp.StatusCode() != http.StatusForbidden {
			t.Errorf("bob down-scoped to other-region: want 403, got %d", resp.StatusCode())
		}

		allowed := scopedEditor("bob", "bob-pass", &resource.TokenScope{Regions: []string{"itbg-bergamo"}})
		client2, err := storagev1.NewClientWithResponses(regionalBaseURL+"/providers/seca.storage", storagev1.WithRequestEditorFn(allowed))
		if err != nil {
			t.Fatalf("create storage client: %v", err)
		}
		resp2, err := client2.ListBlockStoragesWithResponse(context.Background(), testTenant, testWorkspace, &storagev1.ListBlockStoragesParams{})
		if err != nil {
			t.Fatalf("list block storages (in-scope): %v", err)
		}
		if resp2.StatusCode() != http.StatusOK {
			t.Errorf("bob down-scoped to itbg-bergamo: want 200, got %d", resp2.StatusCode())
		}
	})

	t.Run("carol can list workspaces (e2e-workspace-editor multi-subject)", func(t *testing.T) {
		// carol is in ra-multi-subject with dave; both have workspace-editor role.
		editor := identityEditor("carol", "carol-pass")
		client, err := workspacev1sdk.NewClientWithResponses(regionalBaseURL+"/providers/seca.workspace", workspacev1sdk.WithRequestEditorFn(editor))
		if err != nil {
			t.Fatalf("create workspace client: %v", err)
		}
		resp, err := client.ListWorkspacesWithResponse(context.Background(), testTenant, &workspacev1sdk.ListWorkspacesParams{})
		if err != nil {
			t.Fatalf("list workspaces: %v", err)
		}
		if resp.StatusCode() != http.StatusOK {
			t.Errorf("carol list workspaces: want 200, got %d", resp.StatusCode())
		}
	})

	t.Run("dave can list workspaces (e2e-workspace-editor multi-subject)", func(t *testing.T) {
		// dave is in the same multi-subject assignment as carol.
		editor := identityEditor("dave", "dave-pass")
		client, err := workspacev1sdk.NewClientWithResponses(regionalBaseURL+"/providers/seca.workspace", workspacev1sdk.WithRequestEditorFn(editor))
		if err != nil {
			t.Fatalf("create workspace client: %v", err)
		}
		resp, err := client.ListWorkspacesWithResponse(context.Background(), testTenant, &workspacev1sdk.ListWorkspacesParams{})
		if err != nil {
			t.Fatalf("list workspaces: %v", err)
		}
		if resp.StatusCode() != http.StatusOK {
			t.Errorf("dave list workspaces: want 200, got %d", resp.StatusCode())
		}
	})

	t.Run("storage viewer denied workspace write (wrong provider)", func(t *testing.T) {
		// alice's assignment grants only e2e-region-viewer, not e2e-workspace-editor. Denied writing workspace.
		editor := identityEditor("alice", "alice-pass")
		client, err := workspacev1sdk.NewClientWithResponses(regionalBaseURL+"/providers/seca.workspace", workspacev1sdk.WithRequestEditorFn(editor))
		if err != nil {
			t.Fatalf("create workspace client: %v", err)
		}
		ws := testWorkspace + "-alice-attempt"
		resp, err := client.CreateOrUpdateWorkspaceWithResponse(
			context.Background(),
			testTenant,
			ws,
			&workspacev1sdk.CreateOrUpdateWorkspaceParams{},
			schema.Workspace{},
		)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}
		if resp.StatusCode() != http.StatusForbidden {
			t.Errorf("alice write workspace: want 403, got %d", resp.StatusCode())
		}
	})
}
