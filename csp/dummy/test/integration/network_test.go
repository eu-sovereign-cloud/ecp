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
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

func TestNetwork(t *testing.T) {
	t.Parallel()

	t.Run("should create a network resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-net-create-" + uuid.New().String()[:8]
		netDomain := &netdom.Network{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant"},
			},
			Spec: netdom.NetworkSpec{
				CIDR:          netdom.CIDR{IPv4: "10.0.0.0/24"},
				SkuRef:        commondomain.Reference{Resource: "sku-1"},
				RouteTableRef: commondomain.Reference{Resource: "rt-1"},
			},
		}

		_, err := networkRepo.Create(t.Context(), netDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedNet := &netdom.Network{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := networkRepo.Load(ctx, &loadedNet); err != nil {
				return false, err
			}
			return loadedNet.Status != nil && loadedNet.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "network resource should become active")
	})

	t.Run("should delete a network resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-net-delete-" + uuid.New().String()[:8]
		netDomain := &netdom.Network{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant"},
			},
			Spec: netdom.NetworkSpec{
				CIDR:          netdom.CIDR{IPv4: "10.0.1.0/24"},
				SkuRef:        commondomain.Reference{Resource: "sku-1"},
				RouteTableRef: commondomain.Reference{Resource: "rt-1"},
			},
		}

		_, err := networkRepo.Create(t.Context(), netDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedNet := &netdom.Network{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := networkRepo.Load(ctx, &loadedNet); err != nil {
				return false, err
			}
			return loadedNet.Status != nil && loadedNet.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "network resource should become active before deletion")

		err = networkRepo.Delete(t.Context(), netDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedNet := &netdom.Network{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := networkRepo.Load(ctx, &loadedNet); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "network resource should be deleted")
	})
}
