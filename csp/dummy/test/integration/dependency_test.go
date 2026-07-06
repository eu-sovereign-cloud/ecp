//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

func TestBlockStorageImageDependencies(t *testing.T) {
	t.Parallel()

	t.Run("should block block storage deletion while an image references it", func(t *testing.T) {
		t.Parallel()

		// Given a block storage with an image stored on it, both active.
		bsName := "test-dep-block-bs-" + uuid.New().String()[:8]
		createActiveBlockStorage(t, bsName)

		imageName := "test-dep-block-img-" + uuid.New().String()[:8]
		img := createActiveImage(t, imageName, bsName)

		// When the block storage is deleted while the image still references it,
		// the deletion is blocked and a DeletionBlocked condition is reported.
		err := blockStorageRepo.Delete(t.Context(), newBlockStorage(bsName, nil))
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded, err := loadBlockStorage(ctx, bsName)
			if err != nil {
				return false, err
			}
			return loaded.Status != nil && hasConditionType(loaded.Status.Conditions, "DeletionBlocked"), nil
		})
		require.NoError(t, err, "block storage deletion should be blocked while an image references it")

		// And the block storage is still present (not deleted).
		_, err = loadBlockStorage(t.Context(), bsName)
		require.NoError(t, err, "block storage should still exist while deletion is blocked")

		// When the referencing image is deleted, the block storage deletion completes.
		err = imageRepo.Delete(t.Context(), img)
		require.NoError(t, err)
		requireImageDeleted(t, imageName)

		requireBlockStorageDeleted(t, bsName)
	})

	t.Run("should create a block storage from an active source image", func(t *testing.T) {
		t.Parallel()

		// Given an active source image (which itself requires an active block storage).
		sourceBSName := "test-dep-src-bs-" + uuid.New().String()[:8]
		createActiveBlockStorage(t, sourceBSName)

		imageName := "test-dep-src-img-" + uuid.New().String()[:8]
		createActiveImage(t, imageName, sourceBSName)

		// When a block storage is created from that source image, it becomes active.
		sourceImageRef := imageRefFor(imageName)
		clonedBSName := "test-dep-cloned-bs-" + uuid.New().String()[:8]
		_, err := blockStorageRepo.Create(t.Context(), newBlockStorage(clonedBSName, &sourceImageRef))
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, dependencyTimeout, true, func(ctx context.Context) (bool, error) {
			loaded, err := loadBlockStorage(ctx, clonedBSName)
			if err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "block storage created from an active source image should become active")
	})

	t.Run("should keep an image pending while its block storage does not exist", func(t *testing.T) {
		t.Parallel()

		// Given an image referencing a block storage that does not exist.
		imageName := "test-dep-missing-img-" + uuid.New().String()[:8]
		img := newImage(imageName, blockStorageRefFor("missing-bs-"+uuid.New().String()[:8]))
		_, err := imageRepo.Create(t.Context(), img)
		require.NoError(t, err)

		// The image reports a DependencyPending condition and never becomes active.
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded, err := loadImage(ctx, imageName)
			if err != nil {
				return false, err
			}
			return loaded.Status != nil && hasConditionType(loaded.Status.Conditions, "DependencyPending"), nil
		})
		require.NoError(t, err, "image should wait for its block storage to exist")

		loaded, err := loadImage(t.Context(), imageName)
		require.NoError(t, err)
		require.NotEqual(t, commondomain.ResourceStateActive, loaded.Status.State)

		// Cleanup: the image can still be deleted while waiting on its dependency.
		require.NoError(t, imageRepo.Delete(t.Context(), img))
	})
}
