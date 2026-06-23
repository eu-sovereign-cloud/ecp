//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/workspace/v1"
)

func TestWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("should create a workspace resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-ws-create-" + uuid.New().String()[:8]
		wsDomain := &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant"},
			},
			Spec: wsdom.WorkspaceSpec{},
		}

		_, err := workspaceRepo.Create(t.Context(), wsDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedWs := &wsdom.Workspace{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := workspaceRepo.Load(ctx, &loadedWs); err != nil {
				return false, err
			}
			return loadedWs.Status != nil && loadedWs.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "workspace resource should become active")
	})

	t.Run("should delete a workspace resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-ws-delete-" + uuid.New().String()[:8]
		wsDomain := &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant"},
			},
			Spec: wsdom.WorkspaceSpec{},
		}
		_, err := workspaceRepo.Create(t.Context(), wsDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedWs := &wsdom.Workspace{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := workspaceRepo.Load(ctx, &loadedWs); err != nil {
				return false, err
			}
			return loadedWs.Status != nil && loadedWs.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "workspace resource should become active before deletion")

		err = workspaceRepo.Delete(t.Context(), wsDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedWs := &wsdom.Workspace{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := workspaceRepo.Load(ctx, &loadedWs); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "workspace resource should be deleted")
	})
}
