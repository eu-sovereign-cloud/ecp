//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	ecpmodel "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	regionalmodel "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

func TestWorkspaceAPI(t *testing.T) {
	//t.Parallel()

	t.Run("should create a workspace resource via the gateway API", func(t *testing.T) {
		//t.Parallel()

		//
		// Given a unique workspace name and an empty request body
		workspaceName := "test-ws-create-" + uuid.New().String()[:8]
		body, err := json.Marshal(schema.Workspace{})
		require.NoError(t, err)

		//
		// When we call CreateOrUpdateWorkspace on the SDK client
		resp, err := workspaceClient.CreateOrUpdateWorkspaceWithBody(context.Background(), testTenant, workspaceName, nil, "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		_ = resp.Body.Close()

		//
		// Then the API call should return a success status
		require.Equal(t, http.StatusOK, resp.StatusCode)

		//
		// And the workspace custom resource should eventually become active in the cluster
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			var createdWorkspace workspacev1.Workspace
			ns := kubernetesadapter.ComputeNamespace(&scope.Scope{Tenant: testTenant, Workspace: workspaceName})
			key := client.ObjectKey{
				Namespace: ns,
				Name:      workspaceName,
			}
			if err := k8sClient.Get(ctx, key, &createdWorkspace); err != nil {
				return false, nil // Keep retrying if not found
			}

			if createdWorkspace.Status != nil && createdWorkspace.Status.State != nil && regional.ResourceStateDomain(*createdWorkspace.Status.State) == regional.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "workspace CR should become active")

		//
		// And we can cleanup the workspace
		state := regional.ResourceStateDeleting
		wsDomain := &regionalmodel.WorkspaceDomain{
			Metadata: regionalmodel.Metadata{
				CommonMetadata: ecpmodel.CommonMetadata{
					Name: workspaceName,
				},
				Scope: scope.Scope{
					Tenant:    testTenant,
					Workspace: workspaceName,
				},
			},
			Spec: regionalmodel.WorkspaceSpec{},
			Status: &regional.WorkspaceStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: &state,
				},
			},
		}

		_, err = workspaceRepo.Update(t.Context(), wsDomain)
		require.NoError(t, err)
	})

	t.Run("should delete a workspace resource via the gateway API", func(t *testing.T) {
		//t.Parallel()

		//
		// Given a unique workspace that has been created
		workspaceName := "test-ws-delete-" + uuid.New().String()[:8]
		createBody, err := json.Marshal(schema.Workspace{})
		require.NoError(t, err)
		createResp, err := workspaceClient.CreateOrUpdateWorkspaceWithBody(context.Background(), testTenant, workspaceName, nil, "application/json", bytes.NewReader(createBody))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode)
		_ = createResp.Body.Close()

		// And the resource is active in the cluster
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			var createdWorkspace workspacev1.Workspace
			ns := kubernetesadapter.ComputeNamespace(&scope.Scope{Tenant: testTenant, Workspace: workspaceName})
			key := client.ObjectKey{Namespace: ns, Name: workspaceName}
			if err := k8sClient.Get(ctx, key, &createdWorkspace); err != nil {
				return false, nil
			}
			if createdWorkspace.Status != nil && createdWorkspace.Status.State != nil && regional.ResourceStateDomain(*createdWorkspace.Status.State) == regional.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "workspace CR should become active before deletion")

		//
		// When we call DeleteWorkspace on the SDK client
		delResp, err := workspaceClient.DeleteWorkspace(context.Background(), testTenant, workspaceName, nil)
		require.NoError(t, err)
		_ = delResp.Body.Close()

		//
		// Then the API call should return a success status
		require.Equal(t, http.StatusAccepted, delResp.StatusCode)

		//
		// And the workspace custom resource should eventually be marked for deletion in the cluster
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			var createdWorkspace workspacev1.Workspace
			ns := kubernetesadapter.ComputeNamespace(&scope.Scope{Tenant: testTenant, Workspace: workspaceName})
			key := client.ObjectKey{Namespace: ns, Name: workspaceName}
			if err := k8sClient.Get(ctx, key, &createdWorkspace); err != nil {
				if kerrs.IsNotFound(err) {
					return true, nil
				}

				return false, err
			}

			return false, nil
		})
		require.NoError(t, err, "workspace CR should have a deletion timestamp")
	})
}
