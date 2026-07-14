//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

func newTestInstance(name string) *instancedom.Instance {
	inst := &instancedom.Instance{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: name},
			Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
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
	return inst
}

func TestInstance(t *testing.T) {
	t.Parallel()

	t.Run("should create an instance resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-instance-create-" + uuid.New().String()[:8]
		instanceDomain := newTestInstance(resourceName)

		_, err := instanceRepo.Create(t.Context(), instanceDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedInstance := newTestInstance(resourceName)
			if err := instanceRepo.Load(ctx, &loadedInstance); err != nil {
				return false, err
			}
			return loadedInstance.Status != nil && loadedInstance.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "instance resource should become active")
	})

	t.Run("should delete an instance resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-instance-delete-" + uuid.New().String()[:8]
		instanceDomain := newTestInstance(resourceName)

		_, err := instanceRepo.Create(t.Context(), instanceDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedInstance := newTestInstance(resourceName)
			if err := instanceRepo.Load(ctx, &loadedInstance); err != nil {
				return false, err
			}
			return loadedInstance.Status != nil && loadedInstance.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "instance resource should become active before deletion")

		err = instanceRepo.Delete(t.Context(), instanceDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedInstance := newTestInstance(resourceName)
			if err := instanceRepo.Load(ctx, &loadedInstance); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "instance resource should be deleted")
	})
}
