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
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

func TestBlockStorage(t *testing.T) {
	t.Parallel()

	t.Run("should create a block storage resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-bs-create-" + uuid.New().String()[:8]
		bsDomain := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope: kernelresource.Scope{
					Tenant:    "test-tenant",
					Workspace: "test-workspace",
				},
			},
			Spec: bsdom.BlockStorageSpec{
				SizeGB: 1,
				SkuRef: commondomain.Reference{Resource: "sku-1"},
			},
		}

		_, err := blockStorageRepo.Create(t.Context(), bsDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedBs := &bsdom.BlockStorage{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := blockStorageRepo.Load(ctx, &loadedBs); err != nil {
				return false, err
			}
			return loadedBs.Status != nil && loadedBs.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "block storage resource should become active")
	})

	t.Run("should delete a block storage resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-bs-delete-" + uuid.New().String()[:8]
		bsDomain := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope: kernelresource.Scope{
					Tenant:    "test-tenant",
					Workspace: "test-workspace",
				},
			},
			Spec: bsdom.BlockStorageSpec{
				SizeGB: 1,
				SkuRef: commondomain.Reference{Resource: "sku-1"},
			},
		}
		_, err := blockStorageRepo.Create(t.Context(), bsDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedBs := &bsdom.BlockStorage{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := blockStorageRepo.Load(ctx, &loadedBs); err != nil {
				return false, err
			}
			return loadedBs.Status != nil && loadedBs.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "block storage resource should become active before deletion")

		err = blockStorageRepo.Delete(t.Context(), bsDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedBs := &bsdom.BlockStorage{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := blockStorageRepo.Load(ctx, &loadedBs); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "block storage resource should be deleted")
	})

	t.Run("should increase the size of a block storage resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-bs-increase-" + uuid.New().String()[:8]
		bsDomain := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope: kernelresource.Scope{
					Tenant:    "test-tenant",
					Workspace: "test-workspace",
				},
			},
			Spec: bsdom.BlockStorageSpec{
				SizeGB: 1,
				SkuRef: commondomain.Reference{Resource: "sku-1"},
			},
		}
		_, err := blockStorageRepo.Create(t.Context(), bsDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedBs := &bsdom.BlockStorage{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := blockStorageRepo.Load(ctx, &loadedBs); err != nil {
				return false, err
			}
			return loadedBs.Status != nil && loadedBs.Status.State == commondomain.ResourceStateActive && loadedBs.Status.SizeGB == 1, nil
		})
		require.NoError(t, err, "block storage resource should become active with initial size")

		updatedBsDomain := &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
			},
		}
		err = blockStorageRepo.Load(t.Context(), &updatedBsDomain)
		require.NoError(t, err)

		updatedBsDomain.Spec.SizeGB = 2
		_, err = blockStorageRepo.Update(t.Context(), updatedBsDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			currentBs := &bsdom.BlockStorage{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := blockStorageRepo.Load(ctx, &currentBs); err != nil {
				return false, err
			}
			return currentBs.Status != nil && currentBs.Status.State == commondomain.ResourceStateActive && currentBs.Status.SizeGB == 2, nil
		})
		require.NoError(t, err, "block storage resource should have its size increased")
	})
}
