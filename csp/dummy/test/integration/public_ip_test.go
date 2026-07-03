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
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

func TestPublicIp(t *testing.T) {
	t.Parallel()

	t.Run("should create a public ip resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-public-ip-create-" + uuid.New().String()[:8]
		publicIpDomain := &publicipdom.PublicIp{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
			},
			Spec: publicipdom.PublicIpSpec{
				Version: commondomain.IPVersionIPv4,
			},
		}

		_, err := publicIpRepo.Create(t.Context(), publicIpDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedPublicIp := &publicipdom.PublicIp{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := publicIpRepo.Load(ctx, &loadedPublicIp); err != nil {
				return false, err
			}
			return loadedPublicIp.Status != nil && loadedPublicIp.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "public ip resource should become active")
	})

	t.Run("should delete a public ip resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-public-ip-delete-" + uuid.New().String()[:8]
		publicIpDomain := &publicipdom.PublicIp{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
			},
			Spec: publicipdom.PublicIpSpec{
				Version: commondomain.IPVersionIPv4,
			},
		}

		_, err := publicIpRepo.Create(t.Context(), publicIpDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedPublicIp := &publicipdom.PublicIp{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := publicIpRepo.Load(ctx, &loadedPublicIp); err != nil {
				return false, err
			}
			return loadedPublicIp.Status != nil && loadedPublicIp.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "public ip resource should become active before deletion")

		err = publicIpRepo.Delete(t.Context(), publicIpDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedPublicIp := &publicipdom.PublicIp{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
				},
			}
			if err := publicIpRepo.Load(ctx, &loadedPublicIp); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "public ip resource should be deleted")
	})
}
