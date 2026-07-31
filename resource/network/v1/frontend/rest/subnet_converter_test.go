package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

func TestSubnetFromAPIToAPIRoundTrip(t *testing.T) {
	sdk := sdkschema.Subnet{
		Spec: sdkschema.SubnetSpec{
			Cidr: sdkschema.Cidr{Ipv4: "10.0.0.0/24"},
			// Reference.resource: {collection}/{name}
			// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
			RouteTableRef: sdkschema.Reference{Resource: "route-tables/rt1"},
			Zone:          "zone-a",
		},
	}
	id := &SubnetIdentity{name: "sn1", tenant: "t1", workspace: "w1", network: "n1"}

	dom := subnetFromAPI(sdk, id, "r1")
	require.Equal(t, "sn1", dom.Name)
	require.Equal(t, "t1", dom.Tenant)
	require.Equal(t, "w1", dom.Workspace)
	require.Equal(t, "n1", dom.Network)
	require.Equal(t, "r1", dom.Region)
	require.Equal(t, subnetdom.ProviderID, dom.Provider)
	require.Equal(t, "10.0.0.0/24", dom.Spec.Cidr.IPv4)
	require.Equal(t, "route-tables/rt1", dom.Spec.RouteTableRef.Resource)
	require.Equal(t, "zone-a", dom.Spec.Zone)

	out := subnetToAPIWithVerb(http.MethodPut)(dom)
	require.Equal(t, http.MethodPut, out.Metadata.Verb)
	require.Equal(t, "sn1", out.Metadata.Name)
	require.Equal(t, "n1", out.Metadata.Network)
	require.Equal(t, sdkschema.RegionalNetworkResourceMetadataKindResourceKindSubnet, out.Metadata.Kind)
	require.Equal(t, "10.0.0.0/24", out.Spec.Cidr.Ipv4)
	require.Equal(t, "route-tables/rt1", out.Spec.RouteTableRef.Resource)
	// metadata.resource: networks/{network}/{collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "networks/n1/subnets/sn1", out.Metadata.Resource)
	// metadata.ref: {provider}/tenants/{tenant}/workspaces/{workspace}/networks/{network}/{collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "seca.network/v1/tenants/t1/workspaces/w1/networks/n1/subnets/sn1", out.Metadata.Ref)
}

func TestSubnetIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := subnetIteratorToAPI(nil, nil)
	// ResponseMetadata.resource: {collection}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "subnets", iter.Metadata.Resource)
	require.Equal(t, "seca.network/v1", iter.Metadata.Provider)
}

func TestSubnetToAPI_Status(t *testing.T) {
	dom := &subnetdom.Subnet{
		Spec: subnetdom.SubnetSpec{
			Cidr: subnetdom.CIDR{IPv4: "10.0.0.0/24"},
			Zone: "zone-a",
		},
	}
	dom.Name = "sn1"

	out := subnetToAPIWithVerb(http.MethodGet)(dom)
	require.Nil(t, out.Status)

	dom.Status = &subnetdom.SubnetStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
		Cidr:   &subnetdom.CIDR{IPv4: "10.0.0.0/24"},
	}
	out = subnetToAPIWithVerb(http.MethodGet)(dom)
	require.NotNil(t, out.Status)
	require.Equal(t, sdkschema.ResourceStateActive, out.Status.State)
	require.NotNil(t, out.Status.Cidr)
	require.Equal(t, "10.0.0.0/24", out.Status.Cidr.Ipv4)
}

func TestSubnetToAPI_SkuRefOptional(t *testing.T) {
	dom := &subnetdom.Subnet{
		Spec: subnetdom.SubnetSpec{
			Cidr: subnetdom.CIDR{IPv4: "10.0.0.0/24"},
			Zone: "zone-a",
		},
	}
	dom.Name = "sn1"

	out := subnetToAPIWithVerb(http.MethodGet)(dom)
	require.Nil(t, out.Spec.SkuRef)

	// Reference.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	dom.Spec.SkuRef = commondomain.Reference{Resource: "skus/sku1"}
	out = subnetToAPIWithVerb(http.MethodGet)(dom)
	require.NotNil(t, out.Spec.SkuRef)
	require.Equal(t, "skus/sku1", out.Spec.SkuRef.Resource)
}

func TestSubnetListParamsFromAPI_SetsNetwork(t *testing.T) {
	params := subnetListParamsFromAPI(sdknetwork.ListSubnetsParams{}, "t1", "w1", "n1")
	require.Equal(t, "n1", params.Network)
	require.Equal(t, "t1", params.Tenant)
	require.Equal(t, "w1", params.Workspace)
}

func TestNewSubnetWithIdentity_SetsNetwork(t *testing.T) {
	id := &SubnetIdentity{name: "sn1", tenant: "t1", workspace: "w1", network: "n1", resourceVersion: "5"}
	dom := newSubnetWithIdentity(id)
	require.Equal(t, "sn1", dom.Name)
	require.Equal(t, "t1", dom.Tenant)
	require.Equal(t, "w1", dom.Workspace)
	require.Equal(t, "n1", dom.Network)
	require.Equal(t, "5", dom.ResourceVersion)
}
