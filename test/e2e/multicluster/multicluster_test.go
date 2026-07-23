//go:build multicluster

package multicluster

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"

	workspacev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

const (
	activePollInterval = 2 * time.Second
	activeTimeout      = 2 * time.Minute
)

// wantProviders are the four regional providers a registered region must
// advertise — the set the regional gateway actually serves.
var wantProviders = []string{"seca.workspace", "seca.storage", "seca.network", "seca.compute"}

// TestMultiClusterRegistration asserts the cross-cluster join: that the region
// the global control plane advertises is real, reachable, and that driving it
// puts resources in the REGIONAL cluster and nowhere else.
//
// The suite knows only the global endpoint. Every regional call below is made
// against a URL discovered at run time from the region catalog, so a broken
// registration fails here instead of silently passing on a fixture.
func TestMultiClusterRegistration(t *testing.T) {
	ctx := context.Background()

	// Step 1: the global gateway advertises the region, with all four regional
	// providers, at an address that is not cluster-local.
	region := requireRegion(ctx, t)
	providers := providerURLs(t, region)

	t.Run("region advertises every regional provider", func(t *testing.T) {
		for _, name := range wantProviders {
			require.Containsf(t, providers, name, "region %q advertises no %q provider", testRegion, name)
		}
	})

	t.Run("advertised URLs are not cluster-local", func(t *testing.T) {
		for _, name := range wantProviders {
			raw := providers[name]
			u, err := url.Parse(raw)
			require.NoErrorf(t, err, "provider %q has an unparseable URL %q", name, raw)

			host := u.Hostname()
			require.NotEmptyf(t, host, "provider %q has no host in %q", name, raw)
			// The single-cluster fixture advertises the in-cluster Service DNS
			// name, which resolves nowhere outside the cluster. Naming it here
			// gives a clearer failure than the connection error it would cause
			// in the next subtest — which is the real reachability proof.
			require.NotContainsf(t, host, ".svc", "provider %q advertises cluster-local DNS %q", name, raw)
			require.NotEqualf(t, "ecp-regional-gateway-regional", host,
				"provider %q advertises the in-cluster service name %q", name, raw)
			require.NotEmptyf(t, u.Port(), "provider %q advertises no port in %q", name, raw)
		}
	})

	// Step 2: the advertised workspace endpoint actually serves the SECA API.
	// This is the first call that crosses the cluster boundary.
	workspaceClient, err := workspacev1.NewClientWithResponses(
		providers["seca.workspace"],
		workspacev1.WithRequestEditorFn(authEditor),
	)
	require.NoError(t, err, "failed to build a workspace client from the advertised URL")

	t.Cleanup(func() {
		_, _ = workspaceClient.DeleteWorkspaceWithResponse(ctx, testTenant, testWorkspace, nil)
	})

	t.Run("advertised regional endpoint serves the API", func(t *testing.T) {
		resp, err := workspaceClient.ListWorkspacesWithResponse(ctx, testTenant, nil)
		require.NoErrorf(t, err, "advertised regional endpoint %q is unreachable", providers["seca.workspace"])
		require.Equal(t, http.StatusOK, resp.StatusCode())
	})

	// Step 3: a write through the advertised endpoint reconciles to Active,
	// proving the regional cluster's delegator is the one doing the work.
	t.Run("workspace created through the advertised endpoint reconciles to active", func(t *testing.T) {
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

	// Step 4: the payoff. The CR exists in the regional cluster and NOT in the
	// global one — the resource genuinely crossed a cluster boundary, rather
	// than both gateways sharing one API server as in the single-cluster suite.
	t.Run("workspace CR lands in the regional cluster only", func(t *testing.T) {
		regional, err := regionalDyn.Resource(wsk8s.WorkspaceGVR).
			Namespace(metav1.NamespaceAll).
			List(ctx, metav1.ListOptions{})
		require.NoError(t, err, "failed to list workspaces in the regional cluster")
		require.Truef(t, hasWorkspace(regional.Items, testWorkspace),
			"workspace %q not found in the regional cluster", testWorkspace)

		global, err := globalDyn.Resource(wsk8s.WorkspaceGVR).
			Namespace(metav1.NamespaceAll).
			List(ctx, metav1.ListOptions{})
		// The global cluster has the CRDs installed but must hold no workspace:
		// a NotFound here would equally prove the point.
		if apierrors.IsNotFound(err) {
			return
		}
		require.NoError(t, err, "failed to list workspaces in the global cluster")
		require.Falsef(t, hasWorkspace(global.Items, testWorkspace),
			"workspace %q leaked into the global cluster", testWorkspace)
	})
}

// requireRegion fetches the region catalog from the global gateway and returns
// the region under test.
func requireRegion(ctx context.Context, t *testing.T) schema.Region {
	t.Helper()

	resp, err := regionClient.ListRegionsWithResponse(ctx, nil)
	require.NoError(t, err, "global gateway is unreachable")
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotNil(t, resp.JSON200)
	require.NotEmpty(t, resp.JSON200.Items, "global gateway advertises no regions")

	for _, region := range resp.JSON200.Items {
		if region.Metadata != nil && region.Metadata.Name == testRegion {
			return region
		}
	}
	t.Fatalf("region %q not advertised by the global gateway", testRegion)
	return schema.Region{}
}

// providerURLs indexes a region's advertised provider URLs by provider name.
func providerURLs(t *testing.T, region schema.Region) map[string]string {
	t.Helper()

	urls := make(map[string]string, len(region.Spec.Providers))
	for _, p := range region.Spec.Providers {
		urls[p.Name] = strings.TrimSuffix(p.Url, "/")
	}
	return urls
}

func hasWorkspace(items []unstructured.Unstructured, name string) bool {
	for _, item := range items {
		if item.GetName() == name {
			return true
		}
	}
	return false
}

// waitForActive polls get until it reports the resource is Active, failing the
// test if that does not happen within activeTimeout. Mirrors the helper in the
// single-cluster suite (test/e2e/flow_test.go).
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
