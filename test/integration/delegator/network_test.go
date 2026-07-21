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
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

// TestNetwork exercises the Network resource: a seca.network-provider resource that is
// workspace-scoped (NOT network-scoped, unlike subnet/route-table). It is created via
// the repo adapter and reconciled to Active by the delegator's network plugin.
func TestNetwork(t *testing.T) {
	newNetwork := func(name string) *netdom.Network {
		return &netdom.Network{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: name},
				Scope:          resource.Scope{Tenant: testTenant, Workspace: testWorkspace},
			},
			Spec: netdom.NetworkSpec{
				CIDR:   netdom.CIDR{IPv4: "10.10.0.0/24"},
				SkuRef: commondomain.Reference{Resource: "sku-1"},
			},
		}
	}

	t.Run("should create a network resource", func(t *testing.T) {
		resourceName := "test-net-create-" + uuid.New().String()[:8]
		_, err := networkRepo.Create(t.Context(), newNetwork(resourceName))
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newNetwork(resourceName)
			if err := networkRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "network resource should become active")

		require.NoError(t, networkRepo.Delete(t.Context(), newNetwork(resourceName)))
	})

	t.Run("should delete a network resource", func(t *testing.T) {
		resourceName := "test-net-delete-" + uuid.New().String()[:8]
		_, err := networkRepo.Create(t.Context(), newNetwork(resourceName))
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newNetwork(resourceName)
			if err := networkRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "network resource should become active before deletion")

		require.NoError(t, networkRepo.Delete(t.Context(), newNetwork(resourceName)))

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newNetwork(resourceName)
			if err := networkRepo.Load(ctx, &loaded); err != nil {
				if errors.Is(err, kernel.ErrNotFound) {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "network resource should be deleted")
	})
}
