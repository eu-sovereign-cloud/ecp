//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/image/v1"
)

func TestImage(t *testing.T) {
	t.Parallel()

	t.Run("should create an image resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-img-create-" + uuid.New().String()[:8]
		imgDomain := &imgdom.Image{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant"},
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

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedImg := &imgdom.Image{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := imageRepo.Load(ctx, &loadedImg); err != nil {
				return false, err
			}
			return loadedImg.Status != nil && loadedImg.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "image resource should become active")
	})

	t.Run("should delete an image resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-img-delete-" + uuid.New().String()[:8]
		imgDomain := &imgdom.Image{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant"},
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

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedImg := &imgdom.Image{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := imageRepo.Load(ctx, &loadedImg); err != nil {
				return false, err
			}
			return loadedImg.Status != nil && loadedImg.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "image resource should become active before deletion")

		err = imageRepo.Delete(t.Context(), imgDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedImg := &imgdom.Image{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := imageRepo.Load(ctx, &loadedImg); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "image resource should be deleted")
	})
}
