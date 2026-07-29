//go:build integration

package integration

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestImage(t *testing.T) {
	t.Parallel()

	t.Run("should create an image resource", func(t *testing.T) {
		t.Parallel()

		// An image is always stored on a block storage, so create that dependency first.
		bsName := "test-img-create-bs-" + uuid.New().String()[:8]
		createActiveBlockStorage(t, bsName)

		imageName := "test-img-create-" + uuid.New().String()[:8]
		createActiveImage(t, imageName, bsName)
	})

	t.Run("should delete an image resource", func(t *testing.T) {
		t.Parallel()

		bsName := "test-img-delete-bs-" + uuid.New().String()[:8]
		createActiveBlockStorage(t, bsName)

		imageName := "test-img-delete-" + uuid.New().String()[:8]
		img := createActiveImage(t, imageName, bsName)

		err := imageRepo.Delete(t.Context(), img)
		require.NoError(t, err)

		requireImageDeleted(t, imageName)
	})
}
