//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// newImageBody builds the request body for creating an image. It references the
// workspace-scoped source block storage that TestMain provisions.
func newImageBody() schema.Image {
	return schema.Image{
		Spec: schema.ImageSpec{
			BlockStorageRef: schema.Reference{Resource: "block-storages/" + sourceBlockStorage, Workspace: testWorkspace},
			CpuArchitecture: schema.ImageSpecCpuArchitectureAmd64,
			Boot:            schema.ImageSpecBootUEFI,
			Initializer:     schema.ImageSpecInitializerNone,
		},
	}
}

// TestImageAPI exercises the regional gateway's image REST handler. Create,
// read-back, and delete are verified through the gateway's HTTP responses;
// reconciliation to Active is covered by the delegator suite, so this suite needs no
// reconciler.
func TestImageAPI(t *testing.T) {
	t.Run("should create and retrieve an image resource via the gateway API", func(t *testing.T) {
		//
		// Given a unique image name
		resourceName := "test-img-create-" + uuid.New().String()[:8]

		//
		// When we create it through the gateway
		createResp, err := storageClient.CreateOrUpdateImageWithResponse(context.Background(), testTenant, resourceName, nil, newImageBody())
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())

		//
		// Then it can be read back with the spec we sent
		getResp, err := storageClient.GetImageWithResponse(context.Background(), testTenant, resourceName)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode())
		require.NotNil(t, getResp.JSON200)
		require.NotNil(t, getResp.JSON200.Metadata)
		require.Equal(t, resourceName, getResp.JSON200.Metadata.Name)
		require.Equal(t, schema.ImageSpecCpuArchitectureAmd64, getResp.JSON200.Spec.CpuArchitecture)

		//
		// And it can be deleted
		delResp, err := storageClient.DeleteImageWithResponse(context.Background(), testTenant, resourceName, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, delResp.StatusCode())
	})

	t.Run("should delete an image resource via the gateway API", func(t *testing.T) {
		//
		// Given an image that has been created
		resourceName := "test-img-delete-" + uuid.New().String()[:8]
		createResp, err := storageClient.CreateOrUpdateImageWithResponse(context.Background(), testTenant, resourceName, nil, newImageBody())
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())

		//
		// When we delete it, the gateway accepts the request
		delResp, err := storageClient.DeleteImageWithResponse(context.Background(), testTenant, resourceName, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, delResp.StatusCode())
	})
}
