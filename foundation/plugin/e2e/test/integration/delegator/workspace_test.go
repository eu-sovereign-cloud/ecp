//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	ecpmodel "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	regionalmodel "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

func TestWorkspace(t *testing.T) {
	t.Parallel()
	t.Run("should create a workspace resource", func(t *testing.T) {
		t.Parallel()

		//
		// Given a unique workspace domain resource definition
		resourceName := "test-ws-create-" + uuid.New().String()[:8]
		wsDomain := &regionalmodel.WorkspaceDomain{
			Metadata: regionalmodel.Metadata{
				CommonMetadata: ecpmodel.CommonMetadata{
					Name: resourceName,
				},
				Scope: scope.Scope{
					Tenant: "test-tenant",
				},
			},
			Spec: regionalmodel.WorkspaceSpec{},
		}

		//
		// When we create the workspace resource via the adapter
		_, err := workspaceRepo.Create(t.Context(), wsDomain)
		require.NoError(t, err)

		//
		// Then the resource should eventually become active
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedWs := &regionalmodel.WorkspaceDomain{
				Metadata: regionalmodel.Metadata{
					CommonMetadata: ecpmodel.CommonMetadata{
						Name: resourceName,
					},
					Scope: scope.Scope{
						Tenant: "test-tenant",
					},
				},
			}
			if err := workspaceRepo.Load(ctx, &loadedWs); err != nil {
				return false, err
			}
			if loadedWs.Status.State != nil && *loadedWs.Status.State == regionalmodel.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "workspace resource should become active")
	})

	t.Run("should delete a workspace resource", func(t *testing.T) {
		t.Parallel()

		//
		// Given a unique workspace resource that is already created
		resourceName := "test-ws-delete-" + uuid.New().String()[:8]
		wsDomain := &regionalmodel.WorkspaceDomain{
			Metadata: regionalmodel.Metadata{
				CommonMetadata: ecpmodel.CommonMetadata{
					Name: resourceName,
				},
				Scope: scope.Scope{
					Tenant: "test-tenant",
				},
			},
			Spec: regionalmodel.WorkspaceSpec{},
		}
		_, err := workspaceRepo.Create(t.Context(), wsDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedWs := &regionalmodel.WorkspaceDomain{
				Metadata: regionalmodel.Metadata{
					CommonMetadata: ecpmodel.CommonMetadata{
						Name: resourceName,
					},
					Scope: scope.Scope{
						Tenant: "test-tenant",
					},
				},
			}
			if err := workspaceRepo.Load(ctx, &loadedWs); err != nil {
				return false, err
			}
			if loadedWs.Status.State != nil && *loadedWs.Status.State == regionalmodel.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "workspace resource should become active before deletion")

		//
		// When we delete the workspace resource
		err = workspaceRepo.Delete(t.Context(), wsDomain)
		require.NoError(t, err)

		//
		// Then the resource should eventually be removed
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedWs := &regionalmodel.WorkspaceDomain{
				Metadata: regionalmodel.Metadata{
					CommonMetadata: ecpmodel.CommonMetadata{
						Name: resourceName,
					},
					Scope: scope.Scope{
						Tenant: "test-tenant",
					},
				},
			}
			err := workspaceRepo.Load(ctx, &loadedWs)
			if err != nil && errors.Is(err, ecpmodel.ErrNotFound) { // Corrected IsNotFound check
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
