package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	skudom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku"
)

func TestStorageSKUIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := storageSKUIteratorToAPI(nil, nil)
	// ResponseMetadata.resource: {collection}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "skus", iter.Metadata.Resource)
	require.Equal(t, "seca.storage/v1", iter.Metadata.Provider)
}

func TestStorageSKUToAPI_ResourceAndRef(t *testing.T) {
	sku := &skudom.StorageSKU{}
	sku.Name = "sku1"
	sku.Tenant = "t1"
	sku.Provider = skudom.ProviderID

	out := storageSKUToAPI(sku)

	// metadata.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "skus/sku1", out.Metadata.Resource)
	// metadata.ref: {provider}/tenants/{tenant}/{collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "seca.storage/v1/tenants/t1/skus/sku1", out.Metadata.Ref)
}
