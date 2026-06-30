package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
)

func TestImageIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := imageIteratorToAPI(nil, nil)
	require.Equal(t, "images", iter.Metadata.Resource)
	require.Equal(t, "seca.storage/v1", iter.Metadata.Provider)
}

func TestImageToAPI_ResourceAndRef(t *testing.T) {
	img := &imgdom.Image{}
	img.Name = "img1"
	img.Tenant = "t1"
	img.Provider = imgdom.ProviderID

	out := imageToAPI(img)

	require.Equal(t, "image/img1", out.Metadata.Resource)
	require.Equal(t, "seca.storage/v1/tenants/t1/providers/image/img1", out.Metadata.Ref)
}
