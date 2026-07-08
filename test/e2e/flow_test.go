//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	authv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

const (
	activePollInterval = 2 * time.Second
	activeTimeout      = 2 * time.Minute
)

// TestEndToEnd drives the full stack in one run: it creates resources through the
// gateway REST API and asserts they are reconciled to Active by the delegator
// plugin. This is the real end-to-end path — API → gateway → CR → delegator →
// status — that the isolated integration suites deliberately do not cover.
func TestEndToEnd(t *testing.T) {
	ctx := context.Background()
	blockStorageName := "e2e-bs-" + uuid.New().String()[:8]
	roleName := "e2e-role-" + uuid.New().String()[:8]

	// Best-effort teardown of everything this test creates, in reverse order.
	t.Cleanup(func() {
		_, _ = storageClient.DeleteBlockStorageWithResponse(ctx, testTenant, testWorkspace, blockStorageName, nil)
		_, _ = authClient.DeleteRoleWithResponse(ctx, testTenant, roleName, &authv1.DeleteRoleParams{})
		_, _ = workspaceClient.DeleteWorkspaceWithResponse(ctx, testTenant, testWorkspace, nil)
	})

	// Step 1: the global gateway serves the regions provisioned by test-data.
	t.Run("global gateway lists the deployed regions", func(t *testing.T) {
		resp, err := regionClient.ListRegions(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.NoError(t, err)

		var regions regionv1.RegionIterator
		require.NoError(t, json.Unmarshal(body, &regions))
		require.NotEmpty(t, regions.Items, "expected at least one region")

		found := false
		for _, region := range regions.Items {
			if region.Metadata != nil && region.Metadata.Name == testRegion {
				found = true
			}
		}
		require.Truef(t, found, "expected region %q in the list", testRegion)
	})

	// Step 2: creating a workspace through the regional gateway provisions its
	// namespace and reconciles to Active via the delegator's workspace plugin.
	t.Run("workspace created via API reconciles to active", func(t *testing.T) {
		resp, err := workspaceClient.CreateOrUpdateWorkspaceWithResponse(ctx, testTenant, testWorkspace, nil, schema.Workspace{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		waitForActive(t, "workspace", func(ctx context.Context) (schema.ResourceState, bool, error) {
			r, err := workspaceClient.GetWorkspaceWithResponse(ctx, testTenant, testWorkspace)
			if err != nil {
				return "", false, err
			}
			if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
				return "", false, nil
			}
			return r.JSON200.Status.State, true, nil
		})
	})

	// Step 3 (flagship): a block storage created through the regional gateway is
	// reconciled all the way to Active by the delegator's block-storage plugin.
	t.Run("block storage created via API is reconciled to active by the delegator", func(t *testing.T) {
		body := schema.BlockStorage{
			Spec: schema.BlockStorageSpec{
				SizeGB: 1,
				SkuRef: schema.Reference{Resource: "sku-1"},
			},
		}
		resp, err := storageClient.CreateOrUpdateBlockStorageWithResponse(ctx, testTenant, testWorkspace, blockStorageName, nil, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		waitForActive(t, "block storage", func(ctx context.Context) (schema.ResourceState, bool, error) {
			r, err := storageClient.GetBlockStorageWithResponse(ctx, testTenant, testWorkspace, blockStorageName)
			if err != nil {
				return "", false, err
			}
			if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
				return "", false, nil
			}
			return r.JSON200.Status.State, true, nil
		})
	})

	// Step 4: a role created through the global gateway is reconciled to Active by
	// the delegator's role plugin.
	t.Run("role created via API is reconciled to active by the delegator", func(t *testing.T) {
		body := schema.Role{
			Spec: schema.RoleSpec{
				Permissions: []schema.Permission{
					{
						Provider:  "seca.compute",
						Resources: []string{"instances"},
						Verb:      []string{"get", "list"},
					},
				},
			},
		}
		resp, err := authClient.CreateOrUpdateRoleWithResponse(ctx, testTenant, roleName, &authv1.CreateOrUpdateRoleParams{}, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		waitForActive(t, "role", func(ctx context.Context) (schema.ResourceState, bool, error) {
			r, err := authClient.GetRoleWithResponse(ctx, testTenant, roleName)
			if err != nil {
				return "", false, err
			}
			if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
				return "", false, nil
			}
			return r.JSON200.Status.State, true, nil
		})
	})
}

// waitForActive polls get until it reports the resource is Active, failing the
// test if that does not happen within activeTimeout. get returns the current
// state, whether the status is populated yet, and any transport error.
func waitForActive(t *testing.T, what string, get func(context.Context) (schema.ResourceState, bool, error)) {
	t.Helper()
	err := wait.PollUntilContextTimeout(context.Background(), activePollInterval, activeTimeout, true, func(ctx context.Context) (bool, error) {
		state, ready, err := get(ctx)
		if err != nil {
			return false, err
		}
		if !ready {
			return false, nil
		}
		return state == schema.ResourceStateActive, nil
	})
	require.NoErrorf(t, err, "%s did not reach active state within %s", what, activeTimeout)
}
