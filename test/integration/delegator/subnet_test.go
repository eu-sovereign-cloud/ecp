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
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

// TestSubnet exercises the network-scoped Subnet resource: it lives in the per-network
// namespace of testNetwork (provisioned in TestMain) and is reconciled to Active by the
// delegator's subnet plugin. The extra Network scope segment is what distinguishes a
// network-scoped resource from a workspace-scoped one.
func TestSubnet(t *testing.T) {
	newSubnet := func(name string) *subnetdom.Subnet {
		s := &subnetdom.Subnet{
			RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: name},
				},
				Network: testNetwork,
			},
			Spec: subnetdom.SubnetSpec{
				Cidr:          subnetdom.CIDR{IPv4: "10.0.0.0/24"},
				RouteTableRef: commondomain.Reference{Resource: "route-tables/rt1"},
				Zone:          "zone-1",
			},
		}
		s.Tenant = testTenant
		s.Workspace = testWorkspace
		return s
	}

	t.Run("should create a subnet resource", func(t *testing.T) {
		resourceName := "test-subnet-create-" + uuid.New().String()[:8]
		_, err := subnetRepo.Create(t.Context(), newSubnet(resourceName))
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newSubnet(resourceName)
			if err := subnetRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "subnet resource should become active")

		require.NoError(t, subnetRepo.Delete(t.Context(), newSubnet(resourceName)))
	})

	t.Run("should delete a subnet resource", func(t *testing.T) {
		resourceName := "test-subnet-delete-" + uuid.New().String()[:8]
		_, err := subnetRepo.Create(t.Context(), newSubnet(resourceName))
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newSubnet(resourceName)
			if err := subnetRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "subnet resource should become active before deletion")

		require.NoError(t, subnetRepo.Delete(t.Context(), newSubnet(resourceName)))

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newSubnet(resourceName)
			if err := subnetRepo.Load(ctx, &loaded); err != nil {
				if errors.Is(err, kernel.ErrNotFound) {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "subnet resource should be deleted")
	})
}
