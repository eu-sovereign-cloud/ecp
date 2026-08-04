package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

func TestNetworkIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := networkIteratorToAPI(nil, nil)
	// ResponseMetadata.resource: {collection}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "networks", iter.Metadata.Resource)
	require.Equal(t, "seca.network/v1", iter.Metadata.Provider)
}

func TestNetworkToAPI_ResourceAndRef(t *testing.T) {
	n := &netdom.Network{}
	n.Name = "net1"
	n.Tenant = "t1"
	n.Workspace = "w1"
	n.Provider = netdom.ProviderID

	out := networkToAPI(n)

	// metadata.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "networks/net1", out.Metadata.Resource)
	// metadata.ref: {provider}/tenants/{tenant}/workspaces/{workspace}/{collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "seca.network/v1/tenants/t1/workspaces/w1/networks/net1", out.Metadata.Ref)
}
