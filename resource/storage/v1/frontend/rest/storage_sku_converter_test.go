package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	skudom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku"
)

func TestStorageSKUIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := storageSKUIteratorToAPI(nil, nil)
	require.Equal(t, "skus", iter.Metadata.Resource)
	require.Equal(t, "seca.storage/v1", iter.Metadata.Provider)
}

func TestStorageSKUToAPI_ResourceAndRef(t *testing.T) {
	sku := &skudom.StorageSKU{}
	sku.Name = "sku1"
	sku.Tenant = "t1"
	sku.Provider = skudom.ProviderID

	out := storageSKUToAPI(sku)

	require.Equal(t, "storage-sku/sku1", out.Metadata.Resource)
	require.Equal(t, "seca.storage/v1/tenants/t1/providers/storage-sku/sku1", out.Metadata.Ref)
}
