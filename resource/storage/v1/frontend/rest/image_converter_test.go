package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
)

func TestImageIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := imageIteratorToAPI(nil, nil)
	// ResponseMetadata.resource: {collection}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "images", iter.Metadata.Resource)
	require.Equal(t, "seca.storage/v1", iter.Metadata.Provider)
}

func TestImageToAPI_ResourceAndRef(t *testing.T) {
	img := &imgdom.Image{}
	img.Name = "img1"
	img.Tenant = "t1"
	img.Provider = imgdom.ProviderID

	out := imageToAPI(img)

	// metadata.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "images/img1", out.Metadata.Resource)
	// metadata.ref: {provider}/tenants/{tenant}/{collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "seca.storage/v1/tenants/t1/images/img1", out.Metadata.Ref)
}
