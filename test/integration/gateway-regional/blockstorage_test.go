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

// newBlockStorageBody builds the request body for creating/updating a block storage.
func newBlockStorageBody(sizeGB int) schema.BlockStorage {
	return schema.BlockStorage{
		Spec: schema.BlockStorageSpec{
			SizeGB: sizeGB,
			SkuRef: schema.Reference{Resource: "sku-1"},
		},
	}
}

// TestBlockStorageAPI exercises the regional gateway's block storage REST handler.
// Create, read-back, update, and delete are verified purely through the gateway's
// HTTP responses; reconciliation to Active is the delegator's responsibility and is
// covered by the delegator suite, so this suite needs no reconciler.
func TestBlockStorageAPI(t *testing.T) {
	t.Run("should create and retrieve a block storage resource via the gateway API", func(t *testing.T) {
		//
		// Given a unique block storage name
		resourceName := "test-bs-create-" + uuid.New().String()[:8]

		//
		// When we create it through the gateway
		createResp, err := storageClient.CreateOrUpdateBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, resourceName, nil, newBlockStorageBody(1))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())

		//
		// Then it can be read back with the spec we sent
		getResp, err := storageClient.GetBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, resourceName)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode())
		require.NotNil(t, getResp.JSON200)
		require.NotNil(t, getResp.JSON200.Metadata)
		require.Equal(t, resourceName, getResp.JSON200.Metadata.Name)
		require.Equal(t, 1, getResp.JSON200.Spec.SizeGB)
		require.Equal(t, "sku-1", getResp.JSON200.Spec.SkuRef.Resource)

		//
		// And it can be deleted
		delResp, err := storageClient.DeleteBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, resourceName, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, delResp.StatusCode())
	})

	t.Run("should update a block storage resource's size via the gateway API", func(t *testing.T) {
		//
		// Given a block storage created at 1GB
		resourceName := "test-bs-update-" + uuid.New().String()[:8]
		createResp, err := storageClient.CreateOrUpdateBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, resourceName, nil, newBlockStorageBody(1))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())

		//
		// When we update its size to 2GB through the gateway
		updateResp, err := storageClient.CreateOrUpdateBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, resourceName, nil, newBlockStorageBody(2))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, updateResp.StatusCode())

		//
		// Then the stored spec reflects the new size
		getResp, err := storageClient.GetBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, resourceName)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode())
		require.NotNil(t, getResp.JSON200)
		require.Equal(t, 2, getResp.JSON200.Spec.SizeGB)

		delResp, err := storageClient.DeleteBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, resourceName, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, delResp.StatusCode())
	})

	t.Run("should delete a block storage resource via the gateway API", func(t *testing.T) {
		//
		// Given a block storage that has been created
		resourceName := "test-bs-delete-" + uuid.New().String()[:8]
		createResp, err := storageClient.CreateOrUpdateBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, resourceName, nil, newBlockStorageBody(1))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())

		//
		// When we delete it, the gateway accepts the request
		delResp, err := storageClient.DeleteBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, resourceName, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, delResp.StatusCode())
	})
}
