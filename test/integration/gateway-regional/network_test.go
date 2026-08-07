//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/test/internal/testenv"
)

// networkSkuRefResource is the network's skuRef sent as the sku's full URN, the way a
// client holding only that URN sends it. A sku is tenant-scoped, so the scope is stripped
// into the CR's own fields and re-embedded on read — and the provider pair has to come
// back ahead of it, not behind.
const networkSkuRefResource = "seca.network/v1/tenants/" + testTenant + "/skus/network-sku-1"

// newNetworkBody builds the request body for creating a network.
func newNetworkBody(cidr string) schema.Network {
	return schema.Network{
		Spec: schema.NetworkSpec{
			Cidr:   schema.Cidr{Ipv4: cidr},
			SkuRef: schema.Reference{Resource: networkSkuRefResource},
		},
	}
}

// newRouteTableBody builds a minimal route table body used as a network child.
func newRouteTableBody(destinationCIDR string) schema.RouteTable {
	return schema.RouteTable{
		Spec: schema.RouteTableSpec{
			Routes: []schema.RouteSpec{
				{
					DestinationCidrBlock: destinationCIDR,
					TargetRef:            schema.Reference{Resource: "internet-gateways/igw"},
				},
			},
		},
	}
}

// TestNetworkAPI exercises the regional gateway's network REST handler. Create,
// read-back, and delete are verified through the gateway's HTTP responses;
// reconciliation to Active is covered by the delegator suite, so this suite needs no
// reconciler.
func TestNetworkAPI(t *testing.T) {
	t.Run("should create and retrieve a network resource via the gateway API", func(t *testing.T) {
		//
		// Given a unique network name
		networkName := "test-net-create-" + uuid.New().String()[:8]

		//
		// When we create it through the gateway
		createResp, err := networkClient.CreateOrUpdateNetworkWithResponse(
			context.Background(), testTenant, testWorkspace, networkName, nil, newNetworkBody("10.30.0.0/16"),
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())
		require.NotNil(t, createResp.JSON200)
		require.Equal(t, networkSkuRefResource, createResp.JSON200.Spec.SkuRef.Resource,
			"the create response must echo the sku URN unchanged")

		//
		// Then it can be read back with the spec we sent
		getResp, err := networkClient.GetNetworkWithResponse(context.Background(), testTenant, testWorkspace, networkName)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode())
		require.NotNil(t, getResp.JSON200)
		require.NotNil(t, getResp.JSON200.Metadata)
		require.Equal(t, networkName, getResp.JSON200.Metadata.Name)
		require.Equal(t, "10.30.0.0/16", getResp.JSON200.Spec.Cidr.Ipv4)
		require.Equal(t, networkSkuRefResource, getResp.JSON200.Spec.SkuRef.Resource,
			"the sku URN must survive the CR round-trip with its provider pair still in front")

		//
		// And it can be deleted
		delResp, err := networkClient.DeleteNetworkWithResponse(context.Background(), testTenant, testWorkspace, networkName, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, delResp.StatusCode())
	})

	t.Run("should delete a network resource via the gateway API", func(t *testing.T) {
		//
		// Given a network that has been created
		networkName := "test-net-delete-" + uuid.New().String()[:8]
		createResp, err := networkClient.CreateOrUpdateNetworkWithResponse(
			context.Background(), testTenant, testWorkspace, networkName, nil, newNetworkBody("10.31.0.0/16"),
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())

		//
		// When we delete it, the gateway accepts the request
		delResp, err := networkClient.DeleteNetworkWithResponse(context.Background(), testTenant, testWorkspace, networkName, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, delResp.StatusCode())
	})

	t.Run("should reject network deletion when a route table exists", func(t *testing.T) {
		//
		// Given a network that holds a route table
		networkName := "test-net-nonempty-" + uuid.New().String()[:8]
		rtName := "test-rt-block-delete-" + uuid.New().String()[:8]

		createNet, err := networkClient.CreateOrUpdateNetworkWithResponse(
			context.Background(), testTenant, testWorkspace, networkName, nil, newNetworkBody("10.32.0.0/16"),
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createNet.StatusCode())

		createRT, err := networkClient.CreateOrUpdateRouteTableWithResponse(
			context.Background(), testTenant, testWorkspace, networkName, rtName, nil, newRouteTableBody("10.32.0.0/16"),
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createRT.StatusCode())

		t.Cleanup(func() {
			_, _ = networkClient.DeleteRouteTableWithResponse(context.Background(), testTenant, testWorkspace, networkName, rtName, nil)
			testenv.DeleteUntilGone(context.Background(), func() (*http.Response, error) {
				return networkClient.DeleteNetwork(context.Background(), testTenant, testWorkspace, networkName, nil)
			})
		})

		//
		// When we try to delete the network while the route table still exists
		delResp, err := networkClient.DeleteNetworkWithResponse(context.Background(), testTenant, testWorkspace, networkName, nil)
		require.NoError(t, err)

		//
		// Then the gateway refuses with 409 Conflict
		require.Equal(t, http.StatusConflict, delResp.StatusCode())
		require.NotNil(t, delResp.JSON409)

		//
		// And the network is still readable
		getResp, err := networkClient.GetNetworkWithResponse(context.Background(), testTenant, testWorkspace, networkName)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode())
		require.NotNil(t, getResp.JSON200)
		require.Equal(t, networkName, getResp.JSON200.Metadata.Name)
	})
}
