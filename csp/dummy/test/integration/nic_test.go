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
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

func TestNic(t *testing.T) {
	t.Parallel()

	t.Run("should create a nic resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-nic-create-" + uuid.New().String()[:8]
		nicDomain := &nicdom.Nic{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
			},
			Spec: nicdom.NicSpec{
				Addresses: []string{"10.0.0.5"},
				SubnetRef: commondomain.Reference{Resource: "subnet-1"},
			},
		}

		_, err := nicRepo.Create(t.Context(), nicDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedNic := &nicdom.Nic{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := nicRepo.Load(ctx, &loadedNic); err != nil {
				return false, err
			}
			return loadedNic.Status != nil && loadedNic.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "nic resource should become active")
	})

	t.Run("should delete a nic resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-nic-delete-" + uuid.New().String()[:8]
		nicDomain := &nicdom.Nic{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
			},
			Spec: nicdom.NicSpec{
				Addresses: []string{"10.0.1.5"},
				SubnetRef: commondomain.Reference{Resource: "subnet-1"},
			},
		}

		_, err := nicRepo.Create(t.Context(), nicDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedNic := &nicdom.Nic{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := nicRepo.Load(ctx, &loadedNic); err != nil {
				return false, err
			}
			return loadedNic.Status != nil && loadedNic.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "nic resource should become active before deletion")

		err = nicRepo.Delete(t.Context(), nicDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedNic := &nicdom.Nic{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := nicRepo.Load(ctx, &loadedNic); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "nic resource should be deleted")
	})
}
