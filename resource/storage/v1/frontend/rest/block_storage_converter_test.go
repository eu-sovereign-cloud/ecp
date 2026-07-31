package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

func TestBlockStorageIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := blockStorageIteratorToAPI(nil, nil)
	// ResponseMetadata.resource: {collection}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "block-storages", iter.Metadata.Resource)
	require.Equal(t, "seca.storage/v1", iter.Metadata.Provider)
}

func TestBlockStorageToAPI_ResourceAndRef(t *testing.T) {
	bs := &bsdom.BlockStorage{}
	bs.Name = "bs1"
	bs.Tenant = "t1"
	bs.Workspace = "w1"
	bs.Provider = bsdom.ProviderID

	out := blockStorageToAPI(bs)

	// metadata.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "block-storages/bs1", out.Metadata.Resource)
	// metadata.ref: {provider}/tenants/{tenant}/workspaces/{workspace}/{collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "seca.storage/v1/tenants/t1/workspaces/w1/block-storages/bs1", out.Metadata.Ref)
}
