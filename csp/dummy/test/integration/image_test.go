//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	ecpmodel "github.com/eu-sovereign-cloud/ecp/foundation/models"
	regionalmodel "github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/scope"
)

func TestImage(t *testing.T) {
	t.Parallel()

	t.Run("should create an image resource", func(t *testing.T) {
		t.Parallel()

		// Given a unique image domain resource definition
		resourceName := "test-img-create-" + uuid.New().String()[:8]
		imgDomain := &regionalmodel.ImageDomain{
			Metadata: regionalmodel.Metadata{
				CommonMetadata: ecpmodel.CommonMetadata{
					Name: resourceName,
				},
				Scope: scope.Scope{
					Tenant: "test-tenant",
				},
			},
			Spec: regionalmodel.ImageSpecDomain{
				BlockStorageRef: regionalmodel.ReferenceDomain{
					Resource: "block-storage-1",
				},
				CpuArchitecture: "amd64",
				Initializer:     "cloudinit-22",
				Boot:            "BIOS",
			},
		}

		// When we create the image resource via the adapter
		_, err := imageRepo.Create(t.Context(), imgDomain)
		require.NoError(t, err)

		// Then the resource should eventually become active
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedImg := &regionalmodel.ImageDomain{
				Metadata: regionalmodel.Metadata{
					CommonMetadata: ecpmodel.CommonMetadata{
						Name: resourceName,
					},
					Scope: scope.Scope{
						Tenant: "test-tenant",
					},
				},
			}
			if err := imageRepo.Load(ctx, &loadedImg); err != nil {
				return false, err
			}
			if loadedImg.Status != nil && loadedImg.Status.State == regionalmodel.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "image resource should become active")
	})

	t.Run("should delete an image resource", func(t *testing.T) {
		t.Parallel()

		// Given a unique image resource that is already created
		resourceName := "test-img-delete-" + uuid.New().String()[:8]
		imgDomain := &regionalmodel.ImageDomain{
			Metadata: regionalmodel.Metadata{
				CommonMetadata: ecpmodel.CommonMetadata{
					Name: resourceName,
				},
				Scope: scope.Scope{
					Tenant: "test-tenant",
				},
			},
			Spec: regionalmodel.ImageSpecDomain{
				BlockStorageRef: regionalmodel.ReferenceDomain{
					Resource: "block-storage-1",
				},
				CpuArchitecture: "amd64",
				Initializer:     "cloudinit-22",
				Boot:            "BIOS",
			},
		}
		_, err := imageRepo.Create(t.Context(), imgDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedImg := &regionalmodel.ImageDomain{
				Metadata: regionalmodel.Metadata{
					CommonMetadata: ecpmodel.CommonMetadata{
						Name: resourceName,
					},
					Scope: scope.Scope{
						Tenant: "test-tenant",
					},
				},
			}
			if err := imageRepo.Load(ctx, &loadedImg); err != nil {
				return false, err
			}
			if loadedImg.Status != nil && loadedImg.Status.State == regionalmodel.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "image resource should become active before deletion")

		// When we delete the image resource
		err = imageRepo.Delete(t.Context(), imgDomain)
		require.NoError(t, err)

		// Then the resource should eventually be removed
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedImg := &regionalmodel.ImageDomain{
				Metadata: regionalmodel.Metadata{
					CommonMetadata: ecpmodel.CommonMetadata{
						Name: resourceName,
					},
					Scope: scope.Scope{
						Tenant: "test-tenant",
					},
				},
			}
			if err := imageRepo.Load(ctx, &loadedImg); err != nil {
				if errors.Is(err, ecpmodel.ErrNotFound) {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "image resource should be deleted")
	})
}
