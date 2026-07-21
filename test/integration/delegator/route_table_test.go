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
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

// TestRouteTable exercises the network-scoped RouteTable resource: like Subnet it lives
// in the per-network namespace of testNetwork (provisioned in TestMain) and is reconciled
// to Active by the delegator's route-table plugin.
func TestRouteTable(t *testing.T) {
	newRouteTable := func(name string) *routetabledom.RouteTable {
		rt := &routetabledom.RouteTable{
			RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: name},
				},
				Network: testNetwork,
			},
			Spec: routetabledom.RouteTableSpec{
				Routes: []routetabledom.RouteSpec{
					{DestinationCidrBlock: "10.0.0.0/24", TargetRef: commondomain.Reference{Resource: "instances/inst1"}},
				},
			},
		}
		rt.Tenant = testTenant
		rt.Workspace = testWorkspace
		return rt
	}

	t.Run("should create a route table resource", func(t *testing.T) {
		resourceName := "test-rt-create-" + uuid.New().String()[:8]
		_, err := routeTableRepo.Create(t.Context(), newRouteTable(resourceName))
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newRouteTable(resourceName)
			if err := routeTableRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "route table resource should become active")

		require.NoError(t, routeTableRepo.Delete(t.Context(), newRouteTable(resourceName)))
	})

	t.Run("should delete a route table resource", func(t *testing.T) {
		resourceName := "test-rt-delete-" + uuid.New().String()[:8]
		_, err := routeTableRepo.Create(t.Context(), newRouteTable(resourceName))
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newRouteTable(resourceName)
			if err := routeTableRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "route table resource should become active before deletion")

		require.NoError(t, routeTableRepo.Delete(t.Context(), newRouteTable(resourceName)))

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := newRouteTable(resourceName)
			if err := routeTableRepo.Load(ctx, &loaded); err != nil {
				if errors.Is(err, kernel.ErrNotFound) {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "route table resource should be deleted")
	})
}
