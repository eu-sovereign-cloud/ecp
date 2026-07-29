//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

const testNetwork = "test-network"

func TestRouteTable(t *testing.T) {
	t.Parallel()

	t.Run("should create a route table resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-rt-create-" + uuid.New().String()[:8]
		routeTableDomain := &routetabledom.RouteTable{
			RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				},
				Network: testNetwork,
			},
			Spec: routetabledom.RouteTableSpec{
				Routes: []routetabledom.RouteSpec{
					{DestinationCidrBlock: "10.0.0.0/24", TargetRef: commondomain.Reference{Resource: "instances/inst1"}},
				},
			},
		}
		routeTableDomain.Tenant = testTenant
		routeTableDomain.Workspace = testWorkspace

		_, err := routeTableRepo.Create(t.Context(), routeTableDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedRouteTable := &routetabledom.RouteTable{
				RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
					RegionalMetadata: commondomain.RegionalMetadata{
						CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					},
					Network: testNetwork,
				},
			}
			loadedRouteTable.Tenant = testTenant
			loadedRouteTable.Workspace = testWorkspace
			if err := routeTableRepo.Load(ctx, &loadedRouteTable); err != nil {
				return false, err
			}
			return loadedRouteTable.Status != nil && loadedRouteTable.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "route table resource should become active")
	})

	t.Run("should delete a route table resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-rt-delete-" + uuid.New().String()[:8]
		routeTableDomain := &routetabledom.RouteTable{
			RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				},
				Network: testNetwork,
			},
			Spec: routetabledom.RouteTableSpec{
				Routes: []routetabledom.RouteSpec{
					{DestinationCidrBlock: "10.0.1.0/24", TargetRef: commondomain.Reference{Resource: "instances/inst2"}},
				},
			},
		}
		routeTableDomain.Tenant = testTenant
		routeTableDomain.Workspace = testWorkspace

		_, err := routeTableRepo.Create(t.Context(), routeTableDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedRouteTable := &routetabledom.RouteTable{
				RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
					RegionalMetadata: commondomain.RegionalMetadata{
						CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					},
					Network: testNetwork,
				},
			}
			loadedRouteTable.Tenant = testTenant
			loadedRouteTable.Workspace = testWorkspace
			if err := routeTableRepo.Load(ctx, &loadedRouteTable); err != nil {
				return false, err
			}
			return loadedRouteTable.Status != nil && loadedRouteTable.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "route table resource should become active before deletion")

		err = routeTableRepo.Delete(t.Context(), routeTableDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedRouteTable := &routetabledom.RouteTable{
				RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
					RegionalMetadata: commondomain.RegionalMetadata{
						CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					},
					Network: testNetwork,
				},
			}
			loadedRouteTable.Tenant = testTenant
			loadedRouteTable.Workspace = testWorkspace
			if err := routeTableRepo.Load(ctx, &loadedRouteTable); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "route table resource should be deleted")
	})
}
