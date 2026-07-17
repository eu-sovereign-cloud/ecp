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
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

func TestSubnet(t *testing.T) {
	t.Parallel()

	t.Run("should create a subnet resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-subnet-create-" + uuid.New().String()[:8]
		subnetDomain := &subnetdom.Subnet{
			RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				},
				Network: testNetwork,
			},
			Spec: subnetdom.SubnetSpec{
				Cidr:          subnetdom.CIDR{IPv4: "10.0.0.0/24"},
				RouteTableRef: commondomain.Reference{Resource: "route-tables/rt1"},
				Zone:          "zone-1",
			},
		}
		subnetDomain.Tenant = testTenant
		subnetDomain.Workspace = testWorkspace

		_, err := subnetRepo.Create(t.Context(), subnetDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedSubnet := &subnetdom.Subnet{
				RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
					RegionalMetadata: commondomain.RegionalMetadata{
						CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					},
					Network: testNetwork,
				},
			}
			loadedSubnet.Tenant = testTenant
			loadedSubnet.Workspace = testWorkspace
			if err := subnetRepo.Load(ctx, &loadedSubnet); err != nil {
				return false, err
			}
			return loadedSubnet.Status != nil && loadedSubnet.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "subnet resource should become active")
	})

	t.Run("should delete a subnet resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-subnet-delete-" + uuid.New().String()[:8]
		subnetDomain := &subnetdom.Subnet{
			RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				},
				Network: testNetwork,
			},
			Spec: subnetdom.SubnetSpec{
				Cidr:          subnetdom.CIDR{IPv4: "10.0.1.0/24"},
				RouteTableRef: commondomain.Reference{Resource: "route-tables/rt2"},
				Zone:          "zone-1",
			},
		}
		subnetDomain.Tenant = testTenant
		subnetDomain.Workspace = testWorkspace

		_, err := subnetRepo.Create(t.Context(), subnetDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedSubnet := &subnetdom.Subnet{
				RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
					RegionalMetadata: commondomain.RegionalMetadata{
						CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					},
					Network: testNetwork,
				},
			}
			loadedSubnet.Tenant = testTenant
			loadedSubnet.Workspace = testWorkspace
			if err := subnetRepo.Load(ctx, &loadedSubnet); err != nil {
				return false, err
			}
			return loadedSubnet.Status != nil && loadedSubnet.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "subnet resource should become active before deletion")

		err = subnetRepo.Delete(t.Context(), subnetDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedSubnet := &subnetdom.Subnet{
				RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
					RegionalMetadata: commondomain.RegionalMetadata{
						CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					},
					Network: testNetwork,
				},
			}
			loadedSubnet.Tenant = testTenant
			loadedSubnet.Workspace = testWorkspace
			if err := subnetRepo.Load(ctx, &loadedSubnet); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "subnet resource should be deleted")
	})
}
