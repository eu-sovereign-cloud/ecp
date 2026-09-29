//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	storagev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	workspacev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	authhelper "github.com/eu-sovereign-cloud/ecp/test/internal/authhelper"
	"github.com/eu-sovereign-cloud/ecp/test/internal/testenv"
)

// secondRegion is the extra region the test stack's regional gateway serves
// (internal/deploy/gateway-regional/values.yaml). It is reachable only under its region
// path prefix; an unprefixed request is still served as the default region.
const secondRegion = "region-two"

// regionalPathClient returns a workspace client rooted at the gateway's region path prefix,
// i.e. the base URL a client would read off that region's entry in the region catalog.
func regionalPathClient(t *testing.T, region string) *workspacev1.ClientWithResponses {
	t.Helper()
	base := fmt.Sprintf("http://localhost:%d/regions/%s/providers/seca.workspace", regionalLocalPort, region)
	c, err := workspacev1.NewClientWithResponses(base, workspacev1.WithRequestEditorFn(authhelper.AdminEditor()))
	require.NoError(t, err)
	return c
}

// listWorkspaceNames returns every workspace name the client sees in testTenant.
func listWorkspaceNames(t *testing.T, c *workspacev1.ClientWithResponses) []string {
	t.Helper()
	resp, err := c.ListWorkspacesWithResponse(context.Background(), testTenant, nil)
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

// TestMultiRegionRouting covers one gateway deployment serving two regions: the region a
// request is addressed to decides where the resource is placed, what a list returns, and —
// for workspace, whose identity is region-qualified — which resource an item operation
// addresses. Regional resources below workspace still share one tenant/workspace namespace
// across regions, so for those nothing but the region label keeps the two regions apart, on
// lists and item operations alike.
func TestMultiRegionRouting(t *testing.T) {
	secondClient := regionalPathClient(t, secondRegion)
	clients := map[string]*workspacev1.ClientWithResponses{testRegion: workspaceClient, secondRegion: secondClient}

	t.Run("the region prefix decides which region the resource is created in", func(t *testing.T) {
		//
		// Given a workspace created through the second region's path prefix
		name := "mr-ws-" + uuid.New().String()[:8]
		t.Cleanup(func() {
			testenv.DeleteUntilGone(context.Background(), func() (*http.Response, error) {
				return secondClient.DeleteWorkspace(context.Background(), testTenant, name, nil)
			})
		})

		createResp, err := secondClient.CreateOrUpdateWorkspaceWithResponse(context.Background(), testTenant, name, nil, schema.Workspace{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())

		//
		// Then it is stamped with that region, not the gateway's default
		require.NotNil(t, createResp.JSON200)
		require.NotNil(t, createResp.JSON200.Metadata)
		require.Equal(t, secondRegion, createResp.JSON200.Metadata.Region)

		//
		// And only that region's list returns it
		require.Contains(t, listWorkspaceNames(t, secondClient), name)
		require.NotContains(t, listWorkspaceNames(t, workspaceClient), name,
			"a list in the default region must not return another region's resources")
	})

	t.Run("the default region is unchanged by the prefix being available", func(t *testing.T) {
		//
		// Given a workspace created without naming a region
		name := "mr-default-" + uuid.New().String()[:8]
		t.Cleanup(func() {
			testenv.DeleteUntilGone(context.Background(), func() (*http.Response, error) {
				return workspaceClient.DeleteWorkspace(context.Background(), testTenant, name, nil)
			})
		})

		createResp, err := workspaceClient.CreateOrUpdateWorkspaceWithResponse(context.Background(), testTenant, name, nil, schema.Workspace{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())
		require.Equal(t, testRegion, createResp.JSON200.Metadata.Region)

		//
		// Then the second region does not see it
		require.NotContains(t, listWorkspaceNames(t, secondClient), name)
	})

	t.Run("a region-scoped role assignment is enforced per request", func(t *testing.T) {
		//
		// Given bob, whose ra-bob-scoped caps him to the default region
		if !authhelper.AuthEnabled() {
			t.Skip("E2E_AUTH_ENABLED=false: skipping authz assertions")
		}
		bob := authhelper.IdentityEditor("bob", "bob-pass")
		base := fmt.Sprintf("http://localhost:%d", regionalLocalPort)

		allowed, err := storagev1.NewClientWithResponses(base+"/providers/seca.storage", storagev1.WithRequestEditorFn(bob))
		require.NoError(t, err)
		denied, err := storagev1.NewClientWithResponses(base+"/regions/"+secondRegion+"/providers/seca.storage", storagev1.WithRequestEditorFn(bob))
		require.NoError(t, err)

		//
		// When he lists storage in each region
		inRegion, err := allowed.ListBlockStoragesWithResponse(context.Background(), testTenant, testWorkspace, &storagev1.ListBlockStoragesParams{})
		require.NoError(t, err)
		outOfRegion, err := denied.ListBlockStoragesWithResponse(context.Background(), testTenant, testWorkspace, &storagev1.ListBlockStoragesParams{})
		require.NoError(t, err)

		//
		// Then only the region his assignment covers is allowed
		require.Equal(t, http.StatusOK, inRegion.StatusCode())
		require.Equal(t, http.StatusForbidden, outOfRegion.StatusCode(),
			"the region the request is addressed to must reach the authorization claim")
	})

	// Below a workspace the namespace has no region dimension, so an item operation addresses the
	// CR by name alone. Its region label is what confines a GET, PUT or DELETE to the region that
	// owns it — without it the claim above would be checked for one region and the operation
	// carried out on another's resource.
	t.Run("an item operation reaches only its own region's resource", func(t *testing.T) {
		//
		// Given a block storage created through the second region
		base := fmt.Sprintf("http://localhost:%d", regionalLocalPort)
		away, err := storagev1.NewClientWithResponses(base+"/regions/"+secondRegion+"/providers/seca.storage",
			storagev1.WithRequestEditorFn(authhelper.AdminEditor()))
		require.NoError(t, err)
		name := "mr-bs-" + uuid.New().String()[:8]
		t.Cleanup(func() {
			testenv.DeleteUntilGone(context.Background(), func() (*http.Response, error) {
				return away.DeleteBlockStorage(context.Background(), testTenant, testWorkspace, name, nil)
			})
		})

		created, err := away.CreateOrUpdateBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, name, nil, newBlockStorageBody(1))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, created.StatusCode())

		//
		// When the default region reads, writes and deletes it by name
		got, err := storageClient.GetBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, name)
		require.NoError(t, err)
		put, err := storageClient.CreateOrUpdateBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, name, nil, newBlockStorageBody(2))
		require.NoError(t, err)
		del, err := storageClient.DeleteBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, name, nil)
		require.NoError(t, err)

		//
		// Then it is not there to read or delete, and its name is taken
		require.Equal(t, http.StatusNotFound, got.StatusCode())
		require.Equal(t, http.StatusConflict, put.StatusCode(),
			"the create found the name taken, and the update may not take another region's resource over")
		require.Equal(t, http.StatusNotFound, del.StatusCode())

		//
		// And it is untouched: still the second region's, with the spec it was created with
		own, err := away.GetBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, name)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, own.StatusCode())
		require.Equal(t, secondRegion, own.JSON200.Metadata.Region)
		require.Equal(t, 1, own.JSON200.Spec.SizeGB)

		//
		// And bob, whose assignment lets him read block storages in the default region only,
		// reads the default region's own but not this one by naming it through the default
		if !authhelper.AuthEnabled() {
			return
		}
		bob, err := storagev1.NewClientWithResponses(base+"/providers/seca.storage",
			storagev1.WithRequestEditorFn(authhelper.IdentityEditor("bob", "bob-pass")))
		require.NoError(t, err)
		bobOwn, err := bob.GetBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, sourceBlockStorage)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, bobOwn.StatusCode(), "bob may read block storages in his region")
		bobGot, err := bob.GetBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, name)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, bobGot.StatusCode(),
			"a role assignment scoped to one region must not read another region's resource")
	})

	// Issue #396: a workspace is identified by tenant *and* region, so the same name is a
	// different workspace in each region rather than one CR the second write overwrites.
	t.Run("the same workspace name in two regions is two workspaces", func(t *testing.T) {
		//
		// Given one name created in both regions, with a spec that says which is which
		name := "mr-same-" + uuid.New().String()[:8]
		t.Cleanup(func() {
			for _, c := range clients {
				testenv.DeleteUntilGone(context.Background(), func() (*http.Response, error) {
					return c.DeleteWorkspace(context.Background(), testTenant, name, nil)
				})
			}
		})

		for region, client := range clients {
			resp, err := client.CreateOrUpdateWorkspaceWithResponse(context.Background(), testTenant, name, nil,
				schema.Workspace{Spec: map[string]any{"where": region}})
			require.NoError(t, err)
			require.Equalf(t, http.StatusOK, resp.StatusCode(), "creating %q in %q must not collide with the other region", name, region)
		}

		//
		// Then a GET in each region returns that region's own resource
		for region, client := range clients {
			resp, err := client.GetWorkspaceWithResponse(context.Background(), testTenant, name)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())
			require.Equal(t, region, resp.JSON200.Metadata.Region)
			require.Equalf(t, region, resp.JSON200.Spec["where"],
				"a GET in %q resolved the other region's workspace", region)
		}

		//
		// And deleting one region's copy leaves the other standing
		_, err := secondClient.DeleteWorkspace(context.Background(), testTenant, name, nil)
		require.NoError(t, err)

		survivor, err := workspaceClient.GetWorkspaceWithResponse(context.Background(), testTenant, name)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, survivor.StatusCode(),
			"deleting a workspace in one region must not delete the same name in another")
	})

	// The response echoes the region, but the CR is what the list filter, the delegator's region
	// scope and the plugin read, and it carries the region twice: as its field and as its label.
	// Both must be the region the request was addressed to, through a create and an update.
	t.Run("the workspace CR carries the addressed region as field and label", func(t *testing.T) {
		name := "mr-sync-" + uuid.New().String()[:8]
		t.Cleanup(func() {
			for _, c := range clients {
				testenv.DeleteUntilGone(context.Background(), func() (*http.Response, error) {
					return c.DeleteWorkspace(context.Background(), testTenant, name, nil)
				})
			}
		})

		for region, client := range clients {
			for _, step := range []string{"create", "update"} {
				resp, err := client.CreateOrUpdateWorkspaceWithResponse(context.Background(), testTenant, name, nil,
					schema.Workspace{Spec: map[string]any{"step": step}})
				require.NoError(t, err)
				require.Equalf(t, http.StatusOK, resp.StatusCode(), "%s in %q", step, region)

				field, label, err := testenv.WorkspaceRegion(context.Background(), dynClient, testTenant, name, region)
				require.NoError(t, err)
				require.Equalf(t, region, field, "region field after %s in %q", step, region)
				require.Equalf(t, region, label, "region label after %s in %q", step, region)
			}
		}
	})

	t.Run("a region this gateway does not serve is a 404", func(t *testing.T) {
		resp, err := regionalPathClient(t, "no-such-region").
			ListWorkspacesWithResponse(context.Background(), testTenant, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode())
	})
}
