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
	networkName := "e2e-net-" + uuid.New().String()[:8]
	subnetName := "e2e-subnet-" + uuid.New().String()[:8]
	routeTableName := "e2e-rt-" + uuid.New().String()[:8]
	instanceName := "e2e-instance-" + uuid.New().String()[:8]

	// Best-effort teardown of everything this test creates, in reverse order. The
	// network-scoped resources (subnet, route-table) are removed before their network.
	t.Cleanup(func() {
		_, _ = computeClient.DeleteInstanceWithResponse(ctx, testTenant, testWorkspace, instanceName, nil)
		_, _ = networkClient.DeleteSubnetWithResponse(ctx, testTenant, testWorkspace, networkName, subnetName, nil)
		_, _ = networkClient.DeleteRouteTableWithResponse(ctx, testTenant, testWorkspace, networkName, routeTableName, nil)
		_, _ = networkClient.DeleteNetworkWithResponse(ctx, testTenant, testWorkspace, networkName, nil)
		_, _ = storageClient.DeleteBlockStorageWithResponse(ctx, testTenant, testWorkspace, blockStorageName, nil)
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

	// Step 4: a workspace-scoped Network created through the regional gateway. Creating
	// it provisions the network's own namespace (NetworkChildren), which the network-scoped
	// resources below live in — so this step must precede the subnet and route table.
	//
	// Its skuRef is sent as the sku's full URN. A sku is tenant-scoped, so this covers the
	// scope-embed branch the workspace-scoped refs below never reach, and the provider pair
	// must stay at the front of the path on the way back out.
	skuRefResource := "seca.network/v1/tenants/" + testTenant + "/skus/sku-1"
	t.Run("network created via API reconciles to active", func(t *testing.T) {
		body := schema.Network{
			Spec: schema.NetworkSpec{
				Cidr:   schema.Cidr{Ipv4: "10.20.0.0/16"},
				SkuRef: schema.Reference{Resource: skuRefResource},
			},
		}
		resp, err := networkClient.CreateOrUpdateNetworkWithResponse(ctx, testTenant, testWorkspace, networkName, nil, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.Equal(t, skuRefResource, resp.JSON200.Spec.SkuRef.Resource,
			"the create response must echo the sku URN unchanged")

		waitForActive(t, "network", func(ctx context.Context) (schema.ResourceState, bool, error) {
			r, err := networkClient.GetNetworkWithResponse(ctx, testTenant, testWorkspace, networkName)
			if err != nil {
				return "", false, err
			}
			if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
				return "", false, nil
			}
			return r.JSON200.Status.State, true, nil
		})

		got, err := networkClient.GetNetworkWithResponse(ctx, testTenant, testWorkspace, networkName)
		require.NoError(t, err)
		require.Equal(t, skuRefResource, got.JSON200.Spec.SkuRef.Resource,
			"the sku URN must survive the CR round-trip with its provider pair still in front")
	})

	// Step 5 (network-scoped): a Subnet created under the network. Its URN carries the
	// extra networks/{network} segment and it lands in the per-network namespace — the
	// path the workspace-scoped resources above never exercise.
	//
	// Its routeTableRef is sent as the route table's full URN — provider pair, scope and
	// nested path — which is what a client holding only that URN (terraform, which reads
	// it straight out of the route table's metadata.ref) puts in the reference path. The
	// scope segments are stripped into the CR's own fields and re-embedded on read, so
	// every part has to come back in its original position: reordering any of them makes
	// terraform reject the apply with "produced an unexpected new value".
	routeTableRefResource := "seca.network/v1/tenants/" + testTenant + "/workspaces/" + testWorkspace +
		"/networks/" + networkName + "/route-tables/" + routeTableName
	t.Run("subnet (network-scoped) created via API reconciles to active", func(t *testing.T) {
		body := schema.Subnet{
			Spec: schema.SubnetSpec{
				Cidr:          schema.Cidr{Ipv4: "10.20.1.0/24"},
				RouteTableRef: schema.Reference{Resource: routeTableRefResource},
				Zone:          "itbg-1",
			},
		}
		resp, err := networkClient.CreateOrUpdateSubnetWithResponse(ctx, testTenant, testWorkspace, networkName, subnetName, nil, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.Equal(t, routeTableRefResource, resp.JSON200.Spec.RouteTableRef.Resource,
			"the create response must echo the reference path unchanged")

		waitForActive(t, "subnet", func(ctx context.Context) (schema.ResourceState, bool, error) {
			r, err := networkClient.GetSubnetWithResponse(ctx, testTenant, testWorkspace, networkName, subnetName)
			if err != nil {
				return "", false, err
			}
			if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
				return "", false, nil
			}
			return r.JSON200.Status.State, true, nil
		})

		got, err := networkClient.GetSubnetWithResponse(ctx, testTenant, testWorkspace, networkName, subnetName)
		require.NoError(t, err)
		require.Equal(t, routeTableRefResource, got.JSON200.Spec.RouteTableRef.Resource,
			"the reference path must survive the CR round-trip with its scope ahead of the network segment")
	})

	// Step 6 (network-scoped): a RouteTable created under the same network.
	t.Run("route table (network-scoped) created via API reconciles to active", func(t *testing.T) {
		body := schema.RouteTable{
			Spec: schema.RouteTableSpec{
				Routes: []schema.RouteSpec{
					{DestinationCidrBlock: "10.20.0.0/16", TargetRef: schema.Reference{Resource: "internet-gateways/igw"}},
				},
			},
		}
		resp, err := networkClient.CreateOrUpdateRouteTableWithResponse(ctx, testTenant, testWorkspace, networkName, routeTableName, nil, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		waitForActive(t, "route table", func(ctx context.Context) (schema.ResourceState, bool, error) {
			r, err := networkClient.GetRouteTableWithResponse(ctx, testTenant, testWorkspace, networkName, routeTableName)
			if err != nil {
				return "", false, err
			}
			if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
				return "", false, nil
			}
			return r.JSON200.Status.State, true, nil
		})
	})

	// Step 7: a workspace-scoped compute Instance, driven through the seca.compute
	// provider and reconciled by the delegator's instance plugin.
	t.Run("compute instance created via API reconciles to active", func(t *testing.T) {
		body := schema.Instance{
			Spec: schema.InstanceSpec{
				BootVolume: schema.VolumeReference{DeviceRef: schema.Reference{Resource: "block-storages/" + blockStorageName}},
				SkuRef:     schema.Reference{Resource: "sku-1"},
				Zone:       "itbg-1",
			},
		}
		resp, err := computeClient.CreateOrUpdateInstanceWithResponse(ctx, testTenant, testWorkspace, instanceName, nil, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		waitForActive(t, "instance", func(ctx context.Context) (schema.ResourceState, bool, error) {
			r, err := computeClient.GetInstanceWithResponse(ctx, testTenant, testWorkspace, instanceName)
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
