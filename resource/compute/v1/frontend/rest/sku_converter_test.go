package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	skudom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku"
)

func TestInstanceSKUIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := instanceSKUIteratorToAPI(nil, nil)
	// ResponseMetadata.resource: {collection}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "skus", iter.Metadata.Resource)
	require.Equal(t, "seca.compute/v1", iter.Metadata.Provider)
}

func TestInstanceSKUToAPI_ResourceAndRef(t *testing.T) {
	sku := &skudom.InstanceSKU{}
	sku.Name = "sku1"
	sku.Tenant = "t1"
	sku.Provider = skudom.ProviderID
	sku.Spec = skudom.InstanceSKUSpec{VCPU: 2, Ram: 4}

	out := instanceSKUToAPI(sku)

	// metadata.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "skus/sku1", out.Metadata.Resource)
	// metadata.ref: {provider}/tenants/{tenant}/{collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "seca.compute/v1/tenants/t1/skus/sku1", out.Metadata.Ref)
	require.Equal(t, 2, out.Spec.VCPU)
	require.Equal(t, 4, out.Spec.Ram)
}
