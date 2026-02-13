//go:build integration

package integration

import (
	"context"
	"errors"
	"log"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	ecpmodel "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	regionalmodel "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

func TestBlockStorage(t *testing.T) {
	//t.Parallel()

	t.Run("should create a block storage resource", func(t *testing.T) {
		//t.Parallel()

		//
		// Given a unique block storage domain resource definition
		resourceName := "test-bs-create-" + uuid.New().String()[:8]
		bsDomain := &regionalmodel.BlockStorageDomain{
			Metadata: regionalmodel.Metadata{
				CommonMetadata: ecpmodel.CommonMetadata{
					Name: resourceName,
				},
				Scope: scope.Scope{
					Tenant:    testTenant,
					Workspace: testWorkspace,
				},
			},
			Spec: regionalmodel.BlockStorageSpec{
				SizeGB: 1,
				SkuRef: regionalmodel.ReferenceObject{
					Region:   "ITBG-Bergamo",
					Resource: "sku-1",
				},
			},
		}

		//
		// When we create the block storage resource via the adapter
		_, err := blockStorageRepo.Create(t.Context(), bsDomain)
		require.NoError(t, err)

		//
		// Then the resource should eventually become active
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedBs := &regionalmodel.BlockStorageDomain{
				Metadata: regionalmodel.Metadata{
					CommonMetadata: ecpmodel.CommonMetadata{
						Name: resourceName,
					},
					Scope: scope.Scope{
						Tenant:    testTenant,
						Workspace: testWorkspace,
					},
				},
			}
			if err := blockStorageRepo.Load(ctx, &loadedBs); err != nil {
				return false, err
			}
			if loadedBs.Status != nil && loadedBs.Status.State != nil && *loadedBs.Status.State == regionalmodel.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "block storage resource should become active")

		//
		// And we can cleanup the block storage
		err = blockStorageRepo.Delete(t.Context(), bsDomain)
		require.NoError(t, err)
	})

	t.Run("should delete a block storage resource", func(t *testing.T) {
		//t.Parallel()

		//
		// Given a unique block storage resource that is already created
		resourceName := "test-bs-delete-" + uuid.New().String()[:8]
		bsDomain := &regionalmodel.BlockStorageDomain{
			Metadata: regionalmodel.Metadata{
				CommonMetadata: ecpmodel.CommonMetadata{
					Name: resourceName,
				},
				Scope: scope.Scope{
					Tenant:    testTenant,
					Workspace: testWorkspace,
				},
			},
			Spec: regionalmodel.BlockStorageSpec{
				SizeGB: 1,
				SkuRef: regionalmodel.ReferenceObject{
					Region:   "ITBG-Bergamo",
					Resource: "sku-1",
				},
			},
		}
		_, err := blockStorageRepo.Create(t.Context(), bsDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedBs := &regionalmodel.BlockStorageDomain{
				Metadata: regionalmodel.Metadata{
					CommonMetadata: ecpmodel.CommonMetadata{
						Name: resourceName,
					},
					Scope: scope.Scope{
						Tenant:    testTenant,
						Workspace: testWorkspace,
					},
				},
			}
			if err := blockStorageRepo.Load(ctx, &loadedBs); err != nil {
				return false, err
			}
			if loadedBs.Status != nil && loadedBs.Status.State != nil && *loadedBs.Status.State == regionalmodel.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "block storage resource should become active before deletion")

		//
		// When we delete the block storage resource
		err = blockStorageRepo.Delete(t.Context(), bsDomain)
		require.NoError(t, err)

		//
		// Then the resource should eventually be removed
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedBs := &regionalmodel.BlockStorageDomain{
				Metadata: regionalmodel.Metadata{
					CommonMetadata: ecpmodel.CommonMetadata{
						Name: resourceName,
					},
					Scope: scope.Scope{
						Tenant:    testTenant,
						Workspace: testWorkspace,
					},
				},
			}
			err := blockStorageRepo.Load(ctx, &loadedBs)
			if err != nil && errors.Is(err, ecpmodel.ErrNotFound) { // Corrected IsNotFound check
				return true, nil
			}
			if err != nil {
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "block storage resource should be deleted")
	})

	t.Run("should increase the size of a block storage resource", func(t *testing.T) {
		//t.Parallel()

		//
		// Given a unique block storage resource that is already created
		resourceName := "test-bs-increase-" + uuid.New().String()[:8]
		bsDomain := &regionalmodel.BlockStorageDomain{
			Metadata: regionalmodel.Metadata{
				CommonMetadata: ecpmodel.CommonMetadata{
					Name: resourceName,
				},
				Scope: scope.Scope{
					Tenant:    testTenant,
					Workspace: testWorkspace,
				},
			},
			Spec: regionalmodel.BlockStorageSpec{
				SizeGB: 1,
				SkuRef: regionalmodel.ReferenceObject{
					Region:   "ITBG-Bergamo",
					Resource: "sku-1",
				},
			},
		}
		_, err := blockStorageRepo.Create(t.Context(), bsDomain)
		require.NoError(t, err)

		log.Println("---> INIT CREATE BS-INCREASESIZE", "name", resourceName)
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedBs := &regionalmodel.BlockStorageDomain{
				Metadata: regionalmodel.Metadata{
					CommonMetadata: ecpmodel.CommonMetadata{
						Name: resourceName,
					},
					Scope: scope.Scope{
						Tenant:    testTenant,
						Workspace: testWorkspace,
					},
				},
			}

			log.Println("---> DURING Load BS-INCREASESIZE", "name", resourceName, "status", loadedBs.Status)

			if err := blockStorageRepo.Load(ctx, &loadedBs); err != nil {
				return false, err
			}
			log.Println("---> DURING STATUS CHECK BS-INCREASESIZE", "name", resourceName, "status", loadedBs.Status)

			if loadedBs.Status != nil && loadedBs.Status.State != nil && *loadedBs.Status.State == regionalmodel.ResourceStateActive && loadedBs.Status.SizeGB == 1 {
				return true, nil
			}

			log.Println("---> DURING CREATE BS-INCREASESIZE", "name", resourceName, "status", loadedBs.Status)
			return false, nil
		})
		require.NoError(t, err, "block storage resource should become active with initial size")
		log.Println("---> END CREATE BS-INCREASESIZE", "name", resourceName)
		//
		// When we update the block storage resource with an increased size
		updatedBsDomain := &regionalmodel.BlockStorageDomain{
			Metadata: regionalmodel.Metadata{
				CommonMetadata: ecpmodel.CommonMetadata{
					Name: resourceName,
				},
				Scope: scope.Scope{
					Tenant:    testTenant,
					Workspace: testWorkspace,
				},
			},
		}
		err = blockStorageRepo.Load(t.Context(), &updatedBsDomain)
		require.NoError(t, err)

		updatedBsDomain.Spec.SizeGB = 2
		_, err = blockStorageRepo.Update(t.Context(), updatedBsDomain)
		require.NoError(t, err)

		//
		// Then the resource status should eventually reflect the new size
		log.Println("---> PRE UPDATE BS-INCREASESIZE", "name", resourceName)

		var currentBs *regionalmodel.BlockStorageDomain

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			currentBs = &regionalmodel.BlockStorageDomain{
				Metadata: regionalmodel.Metadata{
					CommonMetadata: ecpmodel.CommonMetadata{
						Name: resourceName,
					},
					Scope: scope.Scope{
						Tenant:    testTenant,
						Workspace: testWorkspace,
					},
				},
			}
			if err := blockStorageRepo.Load(ctx, &currentBs); err != nil {
				return false, err
			}
			log.Println("---> DURING UPDATE BS-INCREASESIZE", "name", resourceName, "status", currentBs.Status)

			if currentBs.Status != nil && *currentBs.Status.State == regionalmodel.ResourceStateActive && currentBs.Status.SizeGB == 2 {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "block storage resource should have its size increased")
		log.Println("---> END UPDATE BS-INCREASESIZE", "name", resourceName)
		//
		// And we can cleanup the block storage
		err = blockStorageRepo.Delete(t.Context(), currentBs)
		require.NoError(t, err)
	})
}
