//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	resource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/image/v1"
)

func TestImage(t *testing.T) {
	t.Run("should create an image resource", func(t *testing.T) {
		//
		// Given a unique image domain resource definition
		resourceName := "test-img-create-" + uuid.New().String()[:8]
		imgDomain := &imgdom.Image{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name: resourceName,
				},
				Scope: resource.Scope{
					Tenant: testTenant,
				},
			},
			Spec: imgdom.ImageSpec{
				BlockStorageRef: commondomain.Reference{Resource: "block-storages/source-bs"},
				CpuArchitecture: "amd64",
				Boot:            "UEFI",
				Initializer:     "none",
			},
		}

		//
		// When we create the image resource via the adapter
		_, err := imageRepo.Create(t.Context(), imgDomain)
		require.NoError(t, err)

		//
		// Then the resource should eventually become active
		var loadedImg *imgdom.Image

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedImg = &imgdom.Image{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{
						Name: resourceName,
					},
					Scope: resource.Scope{
						Tenant: testTenant,
					},
				},
			}
			if err := imageRepo.Load(ctx, &loadedImg); err != nil {
				return false, err
			}
			if loadedImg.Status != nil && loadedImg.Status.State == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "image resource should become active")
		require.NotNil(t, loadedImg)
		require.NotNil(t, loadedImg.Status)
		require.Equal(t, commondomain.ResourceStateActive, loadedImg.Status.State)

		//
		// And we can cleanup the image
		err = imageRepo.Delete(t.Context(), imgDomain)
		require.NoError(t, err)
	})

	t.Run("should delete an image resource", func(t *testing.T) {
		//
		// Given a unique image resource that is already created
		resourceName := "test-img-delete-" + uuid.New().String()[:8]
		imgDomain := &imgdom.Image{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name: resourceName,
				},
				Scope: resource.Scope{
					Tenant: testTenant,
				},
			},
			Spec: imgdom.ImageSpec{
				BlockStorageRef: commondomain.Reference{Resource: "block-storages/source-bs"},
				CpuArchitecture: "amd64",
				Boot:            "UEFI",
				Initializer:     "none",
			},
		}
		_, err := imageRepo.Create(t.Context(), imgDomain)
		require.NoError(t, err)

		var loadedImg *imgdom.Image

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedImg = &imgdom.Image{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{
						Name: resourceName,
					},
					Scope: resource.Scope{
						Tenant: testTenant,
					},
				},
			}
			if err := imageRepo.Load(ctx, &loadedImg); err != nil {
				return false, err
			}
			if loadedImg.Status != nil && loadedImg.Status.State == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "image resource should become active before deletion")

		//
		// When we delete the image resource
		err = imageRepo.Delete(t.Context(), imgDomain)
		require.NoError(t, err)

		//
		// Then the resource should eventually be removed
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedImg = &imgdom.Image{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{
						Name: resourceName,
					},
					Scope: resource.Scope{
						Tenant: testTenant,
					},
				},
			}
			err := imageRepo.Load(ctx, &loadedImg)
			if err != nil && errors.Is(err, kernel.ErrNotFound) {
				return true, nil
			}
			if err != nil {
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "image resource should be deleted")
	})
}
