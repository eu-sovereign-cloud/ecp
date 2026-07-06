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
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

func TestInternetGateway(t *testing.T) {
	t.Parallel()

	t.Run("should create an internet gateway resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-ig-create-" + uuid.New().String()[:8]
		internetGatewayDomain := &internetgatewaydom.InternetGateway{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
			},
			Spec: internetgatewaydom.InternetGatewaySpec{
				EgressOnly: true,
			},
		}

		_, err := internetGatewayRepo.Create(t.Context(), internetGatewayDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedInternetGateway := &internetgatewaydom.InternetGateway{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := internetGatewayRepo.Load(ctx, &loadedInternetGateway); err != nil {
				return false, err
			}
			return loadedInternetGateway.Status != nil && loadedInternetGateway.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "internet gateway resource should become active")
	})

	t.Run("should delete an internet gateway resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-ig-delete-" + uuid.New().String()[:8]
		internetGatewayDomain := &internetgatewaydom.InternetGateway{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
			},
			Spec: internetgatewaydom.InternetGatewaySpec{
				EgressOnly: false,
			},
		}

		_, err := internetGatewayRepo.Create(t.Context(), internetGatewayDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedInternetGateway := &internetgatewaydom.InternetGateway{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := internetGatewayRepo.Load(ctx, &loadedInternetGateway); err != nil {
				return false, err
			}
			return loadedInternetGateway.Status != nil && loadedInternetGateway.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "internet gateway resource should become active before deletion")

		err = internetGatewayRepo.Delete(t.Context(), internetGatewayDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedInternetGateway := &internetgatewaydom.InternetGateway{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := internetGatewayRepo.Load(ctx, &loadedInternetGateway); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "internet gateway resource should be deleted")
	})
}
