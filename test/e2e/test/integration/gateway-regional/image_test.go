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
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/image/v1"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/image/v1/backend/kubernetes"
)

// newImageBody is a helper to construct the body for creating/updating an image.
func newImageBody(t *testing.T, boot schema.ImageSpecBoot) schema.Image {
	t.Helper()

	return schema.Image{
		Spec: schema.ImageSpec{
			BlockStorageRef: schema.Reference{Resource: "block-storages/source-bs"},
			CpuArchitecture: schema.ImageSpecCpuArchitectureAmd64,
			Boot:            boot,
			Initializer:     schema.ImageSpecInitializerNone,
		},
	}
}

func TestImageAPI(t *testing.T) {
	t.Run("should create an image resource via the gateway API", func(t *testing.T) {
		//
		// Given a unique image resource definition
		resourceName := "test-img-create-" + uuid.New().String()[:8]
		imageBody := newImageBody(t, schema.ImageSpecBootUEFI)
		body, err := json.Marshal(imageBody)
		require.NoError(t, err)

		//
		// When we call the CreateOrUpdateImage method
		resp, err := storageClient.CreateOrUpdateImageWithBody(context.Background(), testTenant, resourceName, nil, "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		_ = resp.Body.Close()

		//
		// Then the API call should return a success status
		require.Equal(t, http.StatusOK, resp.StatusCode)

		//
		// And the image custom resource should eventually become active in the cluster
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			var createdImg imgk8s.Image
			ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant})
			key := client.ObjectKey{Namespace: ns, Name: resourceName}

			if err := k8sClient.Get(ctx, key, &createdImg); err != nil {
				return false, nil // Keep retrying
			}

			if createdImg.Status != nil && commondomain.ResourceState(createdImg.Status.State) == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "image CR should become active")

		//
		// And we can cleanup the image
		imgDomain := &imgdom.Image{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name: resourceName,
				},
				Scope: resource.Scope{
					Tenant: testTenant,
				},
			},
		}

		err = imageRepo.Delete(t.Context(), imgDomain)
		require.NoError(t, err)
	})

	t.Run("should delete an image resource via the gateway API", func(t *testing.T) {
		//
		// Given a unique image resource that has been created
		resourceName := "test-img-delete-" + uuid.New().String()[:8]
		imageBody := newImageBody(t, schema.ImageSpecBootUEFI)
		createBody, err := json.Marshal(imageBody)
		require.NoError(t, err)

		createResp, err := storageClient.CreateOrUpdateImageWithBody(context.Background(), testTenant, resourceName, nil, "application/json", bytes.NewReader(createBody))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode)
		_ = createResp.Body.Close()

		// And the resource is active in the cluster
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			var createdImg imgk8s.Image
			ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant})
			key := client.ObjectKey{Namespace: ns, Name: resourceName}
			if err := k8sClient.Get(ctx, key, &createdImg); err != nil {
				return false, nil
			}
			if createdImg.Status != nil && commondomain.ResourceState(createdImg.Status.State) == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "image CR should become active before deletion")

		//
		// When we call DeleteImage on the SDK client
		delResp, err := storageClient.DeleteImage(context.Background(), testTenant, resourceName, nil)
		require.NoError(t, err)
		_ = delResp.Body.Close()

		//
		// Then the API call should return a success status
		require.Equal(t, http.StatusAccepted, delResp.StatusCode)

		// And the image custom resource should eventually be removed
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout+2*time.Minute, true, func(ctx context.Context) (bool, error) {
			var createdImg imgk8s.Image
			ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant})
			key := client.ObjectKey{Namespace: ns, Name: resourceName}
			if err := k8sClient.Get(ctx, key, &createdImg); err != nil {
				if kerrors.IsNotFound(err) {
					return true, nil // Successfully deleted
				}

				return false, err // Other error
			}
			return false, nil // Not deleted yet
		})
		require.NoError(t, err, "image CR should be deleted")
	})
}
