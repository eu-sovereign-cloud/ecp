//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// TestInstance exercises the workspace-scoped compute Instance resource, reconciled to
// Active by the delegator's instance plugin. The dummy plugin simulates the lifecycle, so
// the boot volume / SKU references need not point at real resources.
func TestInstance(t *testing.T) {
	newInstance := func(name string) *instancedom.Instance {
		return &instancedom.Instance{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: name},
				Scope:          resource.Scope{Tenant: testTenant, Workspace: testWorkspace},
			},
			Spec: instancedom.InstanceSpec{
				BootVolume: instancedom.VolumeReference{
					DeviceRef: commondomain.Reference{Resource: "block-storage/boot"},
					Type:      "virtio",
				},
				SkuRef: commondomain.Reference{Resource: "sku/small"},
				Zone:   "zone-a",
			},
		}
	}

	t.Run("should create an instance resource", func(t *testing.T) {
		resourceName := "test-instance-create-" + uuid.New().String()[:8]
		_, err := instanceRepo.Create(t.Context(), newInstance(resourceName))
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newInstance(resourceName)
			if err := instanceRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "instance resource should become active")

		require.NoError(t, instanceRepo.Delete(t.Context(), newInstance(resourceName)))
	})

	t.Run("should delete an instance resource", func(t *testing.T) {
		resourceName := "test-instance-delete-" + uuid.New().String()[:8]
		_, err := instanceRepo.Create(t.Context(), newInstance(resourceName))
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newInstance(resourceName)
			if err := instanceRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "instance resource should become active before deletion")

		require.NoError(t, instanceRepo.Delete(t.Context(), newInstance(resourceName)))

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newInstance(resourceName)
			if err := instanceRepo.Load(ctx, &loaded); err != nil {
				if errors.Is(err, kernel.ErrNotFound) {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "instance resource should be deleted")
	})
}
