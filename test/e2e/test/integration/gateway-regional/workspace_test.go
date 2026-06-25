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

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	resource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

func TestWorkspaceAPI(t *testing.T) {
	// t.Parallel()

	t.Run("should create a workspace resource via the gateway API", func(t *testing.T) {
		// t.Parallel()

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
			var createdWorkspace wsk8s.Workspace
			ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant})
			key := client.ObjectKey{
				Namespace: ns,
				Name:      workspaceName,
			}
			if err := k8sClient.Get(ctx, key, &createdWorkspace); err != nil {
				return false, nil // Keep retrying if not found
			}

			if createdWorkspace.Status != nil && commondomain.ResourceState(createdWorkspace.Status.State) == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "workspace CR should become active")

		//
		// And we can cleanup the workspace
		wsDomain := &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name: workspaceName,
				},
				Scope: resource.Scope{
					Tenant: testTenant,
				},
			},
		}

		err = workspaceRepo.Delete(t.Context(), wsDomain)
		require.NoError(t, err)
	})

	t.Run("should delete a workspace resource via the gateway API", func(t *testing.T) {
		// t.Parallel()

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
			var createdWorkspace wsk8s.Workspace
			ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant})
			key := client.ObjectKey{Namespace: ns, Name: workspaceName}
			if err := k8sClient.Get(ctx, key, &createdWorkspace); err != nil {
				return false, nil
			}
			if createdWorkspace.Status != nil && commondomain.ResourceState(createdWorkspace.Status.State) == commondomain.ResourceStateActive {
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
			var createdWorkspace wsk8s.Workspace
			ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant})
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
