package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	rdom "github.com/eu-sovereign-cloud/ecp/resource/region/v1"
)

func TestRegionIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := regionIteratorToAPI(nil, nil)
	// TODO_TEST_238_239
	// require.Equal(t, "regions", iter.Metadata.Resource)
	require.Equal(t, "regions", iter.Metadata.Resource)
	require.Equal(t, "secapi.cloud/v1", iter.Metadata.Provider)
}

func TestRegionToAPI_ResourceAndRef(t *testing.T) {
	r := rdom.Region{}
	r.Name = "r1"

	out := regionToAPI(r, "get")

	// TODO_TEST_238_239
	// require.Equal(t, "regions/r1", out.Metadata.Resource)
	require.Equal(t, "regions/r1", out.Metadata.Resource)
	// TODO_TEST_238_239
	// require.Equal(t, "secapi.cloud/v1/regions/r1", out.Metadata.Ref)
	require.Equal(t, "secapi.cloud/v1/regions/r1", out.Metadata.Ref)
}
