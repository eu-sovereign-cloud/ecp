//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	resource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
)

// newBlockStorageBody is a helper to construct the body for creating/updating block storage.
func newBlockStorageBody(t *testing.T, sizeGB int) schema.BlockStorage {
	t.Helper()

	return schema.BlockStorage{
		Spec: schema.BlockStorageSpec{
			SizeGB: sizeGB,
			SkuRef: schema.Reference{Resource: "sku-1"},
		},
	}
}

func TestBlockStorageAPI(t *testing.T) {
	// t.Parallel()

	t.Run("should create a block storage resource via the gateway API", func(t *testing.T) {
		// t.Parallel()

		//
		// Given a unique block storage resource definition
		resourceName := "test-bs-create-" + uuid.New().String()[:8]
		blockStorageBody := newBlockStorageBody(t, 1)
		body, err := json.Marshal(blockStorageBody)
		require.NoError(t, err)

		//
		// When we call the CreateOrUpdateBlockStorage method
		resp, err := storageClient.CreateOrUpdateBlockStorageWithBody(context.Background(), testTenant, testWorkspace, resourceName, nil, "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		_ = resp.Body.Close()

		//
		// Then the API call should return a success status
		require.Equal(t, http.StatusOK, resp.StatusCode)

		//
		// And the block storage custom resource should eventually become active in the cluster
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			var createdBS bsk8s.BlockStorage
			ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant, Workspace: testWorkspace})
			key := client.ObjectKey{Namespace: ns, Name: resourceName}

			if err := k8sClient.Get(ctx, key, &createdBS); err != nil {
				return false, nil // Keep retrying
			}

			if createdBS.Status != nil && commondomain.ResourceState(createdBS.Status.State) == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "block storage CR should become active")

		//
		// And we can cleanup the block storage
		bsDomain := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name: resourceName,
				},
				Scope: resource.Scope{
					Tenant:    testTenant,
					Workspace: testWorkspace,
				},
			},
		}

		err = blockStorageRepo.Delete(t.Context(), bsDomain)
		require.NoError(t, err)
	})

	t.Run("should delete a block storage resource via the gateway API", func(t *testing.T) {
		// t.Parallel()

		//
		// Given a unique block storage resource that has been created
		resourceName := "test-bs-delete-" + uuid.New().String()[:8]
		blockStorageBody := newBlockStorageBody(t, 1)
		createBody, err := json.Marshal(blockStorageBody)
		require.NoError(t, err)

		createResp, err := storageClient.CreateOrUpdateBlockStorageWithBody(context.Background(), testTenant, testWorkspace, resourceName, nil, "application/json", bytes.NewReader(createBody))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode)
		_ = createResp.Body.Close()

		// And the resource is active in the cluster
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			var createdBS bsk8s.BlockStorage
			ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant, Workspace: testWorkspace})
			key := client.ObjectKey{Namespace: ns, Name: resourceName}
			if err := k8sClient.Get(ctx, key, &createdBS); err != nil {
				return false, nil
			}
			if createdBS.Status != nil && commondomain.ResourceState(createdBS.Status.State) == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "block storage CR should become active before deletion")

		//
		// When we call DeleteBlockStorage on the SDK client
		delResp, err := storageClient.DeleteBlockStorage(context.Background(), testTenant, testWorkspace, resourceName, nil)
		require.NoError(t, err)
		_ = delResp.Body.Close()

		//
		// Then the API call should return a success status
		require.Equal(t, http.StatusAccepted, delResp.StatusCode)

		// And the block storage custom resource should eventually be removed
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout+2*time.Minute, true, func(ctx context.Context) (bool, error) {
			var createdBS bsk8s.BlockStorage
			ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant, Workspace: testWorkspace})
			key := client.ObjectKey{Namespace: ns, Name: resourceName}
			if err := k8sClient.Get(ctx, key, &createdBS); err != nil {
				if kerrors.IsNotFound(err) {
					return true, nil // Successfully deleted
				}

				return false, err // Other error
			}
			return false, nil // Not deleted yet
		})
		require.NoError(t, err, "block storage CR should be deleted")
	})

	t.Run("should increase the size of a block storage resource via the gateway API", func(t *testing.T) {
		// t.Parallel()

		//
		// Given a unique block storage resource that is active with a size of 1GB
		resourceName := "test-bs-increase-" + uuid.New().String()[:8]
		blockStorageBody := newBlockStorageBody(t, 1)
		createBody, err := json.Marshal(blockStorageBody)
		require.NoError(t, err)

		createResp, err := storageClient.CreateOrUpdateBlockStorageWithBody(context.Background(), testTenant, testWorkspace, resourceName, nil, "application/json", bytes.NewReader(createBody))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode)
		_ = createResp.Body.Close()

		var createdBS bsk8s.BlockStorage
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant, Workspace: testWorkspace})
			key := client.ObjectKey{Namespace: ns, Name: resourceName}
			if err := k8sClient.Get(ctx, key, &createdBS); err != nil {
				return false, nil
			}
			if createdBS.Status != nil && commondomain.ResourceState(createdBS.Status.State) == commondomain.ResourceStateActive && createdBS.Status.SizeGB == 1 {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "block storage CR should be active with initial size")

		//
		// When we update the resource with an increased size of 2GB
		updateBodyPayload := newBlockStorageBody(t, 2)
		updateBody, err := json.Marshal(updateBodyPayload)
		require.NoError(t, err)

		updateResp, err := storageClient.CreateOrUpdateBlockStorageWithBody(context.Background(), testTenant, testWorkspace, resourceName, nil, "application/json", bytes.NewReader(updateBody))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, updateResp.StatusCode)
		_ = updateResp.Body.Close()

		//
		// Then the resource status should eventually reflect the new size of 2GB
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			var currentBS bsk8s.BlockStorage
			ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant, Workspace: testWorkspace})
			key := client.ObjectKey{Namespace: ns, Name: resourceName}
			if err := k8sClient.Get(ctx, key, &currentBS); err != nil {
				return false, nil
			}
			if currentBS.Status != nil && commondomain.ResourceState(currentBS.Status.State) == commondomain.ResourceStateActive && currentBS.Status.SizeGB == 2 {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "block storage CR should have its size increased to 2GB")

		//
		// And we can cleanup the block storage
		bsDomain := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name: resourceName,
				},
				Scope: resource.Scope{
					Tenant:    testTenant,
					Workspace: testWorkspace,
				},
			},
		}

		err = blockStorageRepo.Delete(t.Context(), bsDomain)
		require.NoError(t, err)
	})
}
