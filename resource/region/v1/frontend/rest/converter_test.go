package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	rdom "github.com/eu-sovereign-cloud/ecp/resource/region/v1"
)

func TestRegionIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := RegionIteratorToAPI(nil, nil)
	require.Equal(t, "regions", iter.Metadata.Resource)
	require.Equal(t, "secapi.cloud/v1", iter.Metadata.Provider)
}

func TestRegionToAPI_ResourceAndRef(t *testing.T) {
	r := rdom.Region{}
	r.Name = "r1"

	out := regionToAPI(r, "get")

	require.Equal(t, "regions/r1", out.Metadata.Resource)
	require.Equal(t, "secapi.cloud/v1/regions/r1", out.Metadata.Ref)
}
