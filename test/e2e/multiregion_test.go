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
// catalog — and reconciled all the way to the delegator. Routing, list isolation and
// region-scoped authorization are the REST layer's, and are covered by
// test/integration/gateway-regional/multiregion_test.go against this same stack.
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

	// Step 3: the same name in the default region is a *different* workspace (issue #396),
	// and the only place that is proven end to end: two CRs the one delegator reconciles
	// independently, rather than a second write re-stamping the first one's resource.
	shared := "e2e-mr-same-" + uuid.New().String()[:8]
	t.Cleanup(func() {
		for _, c := range []*workspacev1.ClientWithResponses{workspaceClient, secondWorkspace} {
			testenv.DeleteUntilGone(ctx, func() (*http.Response, error) {
				return c.DeleteWorkspace(ctx, testTenant, shared, nil)
			})
		}
	})

	for region, client := range map[string]*workspacev1.ClientWithResponses{
		testRegion:   workspaceClient,
		secondRegion: secondWorkspace,
	} {
		created, err := client.CreateOrUpdateWorkspaceWithResponse(ctx, testTenant, shared, nil,
			schema.Workspace{Spec: map[string]any{"where": region}})
		require.NoError(t, err)
		require.Equalf(t, http.StatusOK, created.StatusCode(), "creating %q in %q must not collide with the other region", shared, region)

		waitForActive(t, "workspace "+shared+" in "+region, func(ctx context.Context) (schema.ResourceState, bool, error) {
			r, err := client.GetWorkspaceWithResponse(ctx, testTenant, shared)
			if err != nil {
				return "", false, err
			}
			if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
				return "", false, nil
			}
			return r.JSON200.Status.State, true, nil
		})

		got, err := client.GetWorkspaceWithResponse(ctx, testTenant, shared)
		require.NoError(t, err)
		require.Equalf(t, region, got.JSON200.Spec["where"], "a GET in %q resolved the other region's workspace", region)
		require.Equal(t, region, got.JSON200.Metadata.Region)
	}
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
