//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	storagev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	workspacev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	authhelper "github.com/eu-sovereign-cloud/ecp/test/internal/authhelper"
	"github.com/eu-sovereign-cloud/ecp/test/internal/testenv"
)

// secondRegion is the extra region the one regional gateway of this stack serves, on top
// of testRegion (see internal/deploy/gateway-regional/values.yaml). It has a Region CR of
// its own in the catalog, advertising the same gateway under its region path prefix.
const secondRegion = "region-two"

// TestMultiRegionEndToEnd is the multi-region counterpart of TestEndToEnd: one gateway
// deployment, two regions, discovered the way a client discovers them — off the region
// catalog — and reconciled all the way to the delegator.
func TestMultiRegionEndToEnd(t *testing.T) {
	ctx := context.Background()

	// Step 1: the catalog is what tells a client where a region's providers live, so the
	// second region's base URL is read from it rather than assembled here.
	providerPath := regionProviderPath(t, ctx, secondRegion, "seca.workspace")
	require.True(t, strings.HasPrefix(providerPath, "/regions/"+secondRegion+"/"),
		"the catalog must advertise the second region under its region path prefix, got %q", providerPath)

	// Only the host is swapped for the port-forward: the advertised host is in-cluster DNS,
	// unreachable from the test process, while the path is exactly what the catalog says.
	secondWorkspace, err := workspacev1.NewClientWithResponses(
		regionalURL+providerPath, workspacev1.WithRequestEditorFn(authhelper.AdminEditor()))
	require.NoError(t, err)

	name := "e2e-mr-ws-" + uuid.New().String()[:8]
	t.Cleanup(func() {
		testenv.DeleteUntilGone(ctx, func() (*http.Response, error) {
			return secondWorkspace.DeleteWorkspace(ctx, testTenant, name, nil)
		})
	})

	// Step 2: a resource created through the second region is stamped with it and
	// reconciles to Active — the delegator serves both regions from the one cluster.
	t.Run("a workspace created in the second region reconciles", func(t *testing.T) {
		resp, err := secondWorkspace.CreateOrUpdateWorkspaceWithResponse(ctx, testTenant, name, nil, schema.Workspace{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.NotNil(t, resp.JSON200)
		require.Equal(t, secondRegion, resp.JSON200.Metadata.Region)

		waitForActive(t, "workspace "+name, func(ctx context.Context) (schema.ResourceState, bool, error) {
			r, err := secondWorkspace.GetWorkspaceWithResponse(ctx, testTenant, name)
			if err != nil {
				return "", false, err
			}
			if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
				return "", false, nil
			}
			return r.JSON200.Status.State, true, nil
		})
	})

	// Step 3: the two regions share a tenant namespace, so the region label is the only
	// thing keeping one region's list out of the other's.
	t.Run("the regions do not see each other's resources", func(t *testing.T) {
		require.Contains(t, workspaceNames(t, ctx, secondWorkspace), name)
		require.NotContains(t, workspaceNames(t, ctx, workspaceClient), name)
	})

	// Step 4: the region reaching the authorization claim is what makes a region-scoped
	// RoleAssignment mean anything on a gateway that serves several regions. bob's
	// ra-bob-scoped caps him to testRegion.
	t.Run("a region-scoped role assignment is enforced per request", func(t *testing.T) {
		if !authhelper.AuthEnabled() {
			t.Skip("E2E_AUTH_ENABLED=false: skipping authz assertions")
		}
		bob := authhelper.IdentityEditor("bob", "bob-pass")

		allowed, err := storagev1.NewClientWithResponses(
			regionalURL+"/providers/seca.storage", storagev1.WithRequestEditorFn(bob))
		require.NoError(t, err)
		inRegion, err := allowed.ListBlockStoragesWithResponse(ctx, testTenant, testWorkspace, &storagev1.ListBlockStoragesParams{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, inRegion.StatusCode(), "bob is scoped to "+testRegion)

		denied, err := storagev1.NewClientWithResponses(
			regionalURL+"/regions/"+secondRegion+"/providers/seca.storage", storagev1.WithRequestEditorFn(bob))
		require.NoError(t, err)
		outOfRegion, err := denied.ListBlockStoragesWithResponse(ctx, testTenant, testWorkspace, &storagev1.ListBlockStoragesParams{})
		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, outOfRegion.StatusCode(),
			"the region the request is addressed to must reach the authorization claim")
	})
}

// regionProviderPath returns the URL path the region catalog advertises for one provider
// of a region — the part of the base URL a SECA client is meant to take from the Region CR.
func regionProviderPath(t *testing.T, ctx context.Context, region, provider string) string {
	t.Helper()
	resp, err := regionClient.GetRegionWithResponse(ctx, region)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotNil(t, resp.JSON200)

	for _, p := range resp.JSON200.Spec.Providers {
		if p.Name == provider {
			u, err := url.Parse(p.Url)
			require.NoError(t, err)
			return u.Path
		}
	}
	t.Fatalf("region %q advertises no %q provider", region, provider)
	return ""
}

// workspaceNames returns every workspace name a client sees in testTenant.
func workspaceNames(t *testing.T, ctx context.Context, c *workspacev1.ClientWithResponses) []string {
	t.Helper()
	resp, err := c.ListWorkspacesWithResponse(ctx, testTenant, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotNil(t, resp.JSON200)

	names := make([]string, 0, len(resp.JSON200.Items))
	for _, item := range resp.JSON200.Items {
		if item.Metadata != nil {
			names = append(names, item.Metadata.Name)
		}
	}
	return names
}
