package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

func TestRouteTableFromAPIToAPIRoundTrip(t *testing.T) {
	sdk := sdkschema.RouteTable{
		Spec: sdkschema.RouteTableSpec{
			Routes: []sdkschema.RouteSpec{
				{
					DestinationCidrBlock: "10.0.0.0/24",
					// Reference.resource: {collection}/{name}
					// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
					TargetRef: sdkschema.Reference{Resource: "instances/inst1"},
				},
			},
		},
	}
	id := &RouteTableIdentity{name: "rt1", tenant: "t1", workspace: "w1", network: "n1"}

	dom, err := routeTableFromAPI(sdk, id, "r1")
	require.NoError(t, err)
	require.Equal(t, "rt1", dom.Name)
	require.Equal(t, "t1", dom.Tenant)
	require.Equal(t, "w1", dom.Workspace)
	require.Equal(t, "n1", dom.Network)
	require.Equal(t, "r1", dom.Region)
	require.Equal(t, routetabledom.ProviderID, dom.Provider)
	require.Len(t, dom.Spec.Routes, 1)
	require.Equal(t, "10.0.0.0/24", dom.Spec.Routes[0].DestinationCidrBlock)
	require.Equal(t, "instances/inst1", dom.Spec.Routes[0].TargetRef.Resource)

	out := routeTableToAPIWithVerb(http.MethodPut)(dom)
	require.Equal(t, http.MethodPut, out.Metadata.Verb)
	require.Equal(t, "rt1", out.Metadata.Name)
	require.Equal(t, "n1", out.Metadata.Network)
	require.Equal(t, sdkschema.RegionalNetworkResourceMetadataKindResourceKindRoutingTable, out.Metadata.Kind)
	require.Len(t, out.Spec.Routes, 1)
	require.Equal(t, "10.0.0.0/24", out.Spec.Routes[0].DestinationCidrBlock)
	// metadata.resource: networks/{network}/{collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "networks/n1/route-tables/rt1", out.Metadata.Resource)
	// metadata.ref: {provider}/tenants/{tenant}/workspaces/{workspace}/networks/{network}/{collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "seca.network/v1/tenants/t1/workspaces/w1/networks/n1/route-tables/rt1", out.Metadata.Ref)
}

func TestRouteTableIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := routeTableIteratorToAPI(nil, nil)
	// ResponseMetadata.resource: {collection}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "route-tables", iter.Metadata.Resource)
	require.Equal(t, "seca.network/v1", iter.Metadata.Provider)
}

func TestRouteTableToAPI_Status(t *testing.T) {
	dom := &routetabledom.RouteTable{
		Spec: routetabledom.RouteTableSpec{
			Routes: []routetabledom.RouteSpec{{DestinationCidrBlock: "10.0.0.0/24"}},
		},
	}
	dom.Name = "rt1"

	out := routeTableToAPIWithVerb(http.MethodGet)(dom)
	require.Nil(t, out.Status)

	dom.Status = &routetabledom.RouteTableStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
		Routes: []routetabledom.RouteStatus{{State: commondomain.ResourceStateActive}},
	}
	out = routeTableToAPIWithVerb(http.MethodGet)(dom)
	require.NotNil(t, out.Status)
	require.Equal(t, sdkschema.ResourceStateActive, out.Status.State)
	require.Len(t, out.Status.Routes, 1)
	require.Equal(t, sdkschema.ResourceStateActive, out.Status.Routes[0].State)
}

func TestRouteTableListParamsFromAPI_SetsNetwork(t *testing.T) {
	params := routeTableListParamsFromAPI(sdknetwork.ListRouteTablesParams{}, "t1", "w1", "n1")
	require.Equal(t, "n1", params.Network)
	require.Equal(t, "t1", params.Tenant)
	require.Equal(t, "w1", params.Workspace)
}

func TestNewRouteTableWithIdentity_SetsNetwork(t *testing.T) {
	id := &RouteTableIdentity{name: "rt1", tenant: "t1", workspace: "w1", network: "n1", resourceVersion: "5"}
	dom := newRouteTableWithIdentity(id)
	require.Equal(t, "rt1", dom.Name)
	require.Equal(t, "t1", dom.Tenant)
	require.Equal(t, "w1", dom.Workspace)
	require.Equal(t, "n1", dom.Network)
	require.Equal(t, "5", dom.ResourceVersion)
}
