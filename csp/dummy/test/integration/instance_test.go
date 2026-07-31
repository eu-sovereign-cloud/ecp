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
				// TODO_TEST_238_239
				// DeviceRef: commondomain.Reference{Resource: "block-storage/boot"},
				DeviceRef: commondomain.Reference{Resource: "block-storages/boot"},
				Type:      "virtio",
			},
			// TODO_TEST_238_239
			// SkuRef: commondomain.Reference{Resource: "sku/small"},
			SkuRef: commondomain.Reference{Resource: "skus/small"},
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

	t.Run("should power an instance on, off, and restart", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-instance-power-" + uuid.New().String()[:8]
		instanceDomain := newTestInstance(resourceName)

		_, err := instanceRepo.Create(t.Context(), instanceDomain)
		require.NoError(t, err)
		waitForInstancePowerState(t, resourceName, instancedom.PowerStateOff) // starts powered off
		waitForInstanceState(t, resourceName, commondomain.ResourceStateActive)

		// Start.
		setInstancePowerIntent(t, resourceName, func(inst *instancedom.Instance) {
			inst.DesiredPowerState = instancedom.PowerStateOn
		})
		waitForInstancePowerState(t, resourceName, instancedom.PowerStateOn)

		// Stop.
		setInstancePowerIntent(t, resourceName, func(inst *instancedom.Instance) {
			inst.DesiredPowerState = instancedom.PowerStateOff
		})
		waitForInstancePowerState(t, resourceName, instancedom.PowerStateOff)

		// Start again so we can restart.
		setInstancePowerIntent(t, resourceName, func(inst *instancedom.Instance) {
			inst.DesiredPowerState = instancedom.PowerStateOn
		})
		waitForInstancePowerState(t, resourceName, instancedom.PowerStateOn)

		// Restart: the phased power-off -> power-on cycle ends powered on and clears the
		// restart annotations.
		setInstancePowerIntent(t, resourceName, func(inst *instancedom.Instance) {
			inst.RestartID = "restart-1"
			inst.RestartPhase = instancedom.RestartPhasePowerOff
		})
		waitForInstancePowerState(t, resourceName, instancedom.PowerStateOn)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newTestInstance(resourceName)
			if err := instanceRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.RestartID == "" && loaded.RestartPhase == "", nil
		})
		require.NoError(t, err, "restart annotations should be cleared after the cycle completes")
	})
}

// setInstancePowerIntent loads the instance, applies mutate (setting a power intent field),
// and persists it via Update.
func setInstancePowerIntent(t *testing.T, name string, mutate func(*instancedom.Instance)) {
	t.Helper()
	loaded := newTestInstance(name)
	require.NoError(t, instanceRepo.Load(t.Context(), &loaded))
	mutate(loaded)
	_, err := instanceRepo.Update(t.Context(), loaded)
	require.NoError(t, err)
}

func waitForInstancePowerState(t *testing.T, name string, want instancedom.PowerState) {
	t.Helper()
	err := wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		loaded := newTestInstance(name)
		if err := instanceRepo.Load(ctx, &loaded); err != nil {
			return false, err
		}
		return loaded.Status != nil && loaded.Status.PowerState == want, nil
	})
	require.NoErrorf(t, err, "instance should reach power state %q", want)
}

func waitForInstanceState(t *testing.T, name string, want commondomain.ResourceState) {
	t.Helper()
	err := wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		loaded := newTestInstance(name)
		if err := instanceRepo.Load(ctx, &loaded); err != nil {
			return false, err
		}
		return loaded.Status != nil && loaded.Status.State == want, nil
	})
	require.NoErrorf(t, err, "instance should reach state %q", want)
}
