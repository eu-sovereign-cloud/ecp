package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	skudom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"
)

func TestNetworkSKUIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := networkSKUIteratorToAPI(nil, nil)
	require.Equal(t, "network-skus", iter.Metadata.Resource)
	require.Equal(t, "seca.network/v1", iter.Metadata.Provider)
	require.Equal(t, "list", iter.Metadata.Verb)
	require.Nil(t, iter.Metadata.SkipToken)
}

func TestNetworkSKUIteratorToAPI_SkipToken(t *testing.T) {
	token := "next"
	iter := networkSKUIteratorToAPI([]*skudom.NetworkSKU{newNetworkSKU()}, &token)

	require.Len(t, iter.Items, 1)
	require.NotNil(t, iter.Metadata.SkipToken)
	require.Equal(t, token, *iter.Metadata.SkipToken)
}

func TestNetworkSKUToAPI_ResourceAndRef(t *testing.T) {
	out := networkSKUToAPI(newNetworkSKU())

	require.Equal(t, "network-sku/sku1", out.Metadata.Resource)
	require.Equal(t, "seca.network/v1/tenants/t1/providers/network-sku/sku1", out.Metadata.Ref)
	require.Equal(t, "r1", out.Metadata.Region)
	require.Equal(t, 1000, out.Spec.Bandwidth)
	require.Equal(t, 500, out.Spec.Packets)
}

func newNetworkSKU() *skudom.NetworkSKU {
	sku := &skudom.NetworkSKU{
		Spec: skudom.NetworkSKUSpec{Bandwidth: 1000, Packets: 500},
	}
	sku.Name = "sku1"
	sku.Tenant = "t1"
	sku.Region = "r1"
	sku.Provider = skudom.ProviderID
	return sku
}
