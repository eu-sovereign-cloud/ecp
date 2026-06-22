//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	resource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1/domain"
)

func TestWorkspace(t *testing.T) {
	//t.Parallel()
	t.Run("should create a workspace resource", func(t *testing.T) {
		//t.Parallel()

		//
		// Given a unique workspace domain resource definition
		workspaceName := "test-ws-create-" + uuid.New().String()[:8]
		wsDomain := &wsdom.WorkspaceDomain{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name: workspaceName,
				},
				Scope: resource.Scope{
					Tenant: testTenant,
				},
			},
		}

		//
		// When we create the workspace resource via the adapter
		_, err := workspaceRepo.Create(t.Context(), wsDomain)
		require.NoError(t, err)

		//
		// Then the resource should eventually become active
		var loadedWs *wsdom.WorkspaceDomain
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedWs = &wsdom.WorkspaceDomain{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{
						Name: workspaceName,
					},
					Scope: resource.Scope{
						Tenant: testTenant,
					},
				},
			}
			if err := workspaceRepo.Load(ctx, &loadedWs); err != nil {
				return false, err
			}
			if loadedWs.Status != nil && loadedWs.Status.State == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "workspace resource should become active")
		require.NotNil(t, loadedWs)
		require.NotNil(t, loadedWs.Status)
		require.NotNil(t, loadedWs.Status.State)
		require.Equal(t, commondomain.ResourceStateActive, loadedWs.Status.State)
		//
		// And we can cleanup the workspace
		err = workspaceRepo.Delete(t.Context(), wsDomain)
		require.NoError(t, err)
	})

	t.Run("should delete a workspace resource", func(t *testing.T) {
		//t.Parallel()

		//
		// Given a unique workspace resource that is already created
		workspaceName := "test-ws-delete-" + uuid.New().String()[:8]
		wsDomain := &wsdom.WorkspaceDomain{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name: workspaceName,
				},
				Scope: resource.Scope{
					Tenant: testTenant,
				},
			},
		}
		_, err := workspaceRepo.Create(t.Context(), wsDomain)
		require.NoError(t, err)

		var loadedWs *wsdom.WorkspaceDomain
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedWs = &wsdom.WorkspaceDomain{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{
						Name: workspaceName,
					},
					Scope: resource.Scope{
						Tenant: testTenant,
					},
				},
			}
			if err := workspaceRepo.Load(ctx, &loadedWs); err != nil {
				return false, err
			}
			if loadedWs.Status != nil && loadedWs.Status.State == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "workspace resource should become active before deletion")
		require.NotNil(t, loadedWs)
		require.NotNil(t, loadedWs.Status)
		require.NotNil(t, loadedWs.Status.State)
		require.Equal(t, commondomain.ResourceStateActive, loadedWs.Status.State)
		//
		// When we delete the workspace resource
		err = workspaceRepo.Delete(t.Context(), wsDomain)
		require.NoError(t, err)

		//
		// Then the resource should eventually be removed
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedWs = &wsdom.WorkspaceDomain{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{
						Name: workspaceName,
					},
					Scope: resource.Scope{
						Tenant: testTenant,
					},
				},
			}
			err := workspaceRepo.Load(ctx, &loadedWs)
			if err != nil && errors.Is(err, kernel.ErrNotFound) { // Corrected IsNotFound check
				return true, nil
			}
			if err != nil {
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "workspace resource should be deleted")

	})
}
