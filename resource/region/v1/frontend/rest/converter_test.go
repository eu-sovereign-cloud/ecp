package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	rdom "github.com/eu-sovereign-cloud/ecp/resource/region/v1"
)

func TestRegionIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := regionIteratorToAPI(nil, nil)
	// ResponseMetadata.resource: {collection}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "regions", iter.Metadata.Resource)
	require.Equal(t, "secapi.cloud/v1", iter.Metadata.Provider)
}

func TestRegionToAPI_ResourceAndRef(t *testing.T) {
	r := rdom.Region{}
	r.Name = "r1"

	out := regionToAPI(r, "get")

	// metadata.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "regions/r1", out.Metadata.Resource)
	// metadata.ref: {provider}/{collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "secapi.cloud/v1/regions/r1", out.Metadata.Ref)
}
