package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

func TestBlockStorageIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := BlockStorageIteratorToAPI(nil, nil)
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

	require.Equal(t, "block-storage/bs1", out.Metadata.Resource)
	require.Equal(t, "seca.storage/v1/tenants/t1/workspaces/w1/providers/block-storage/bs1", out.Metadata.Ref)
}
