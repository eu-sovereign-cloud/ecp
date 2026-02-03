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
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

func TestWorkspaceAPI(t *testing.T) {
	t.Parallel()

	// Given: The regional gateway is running and clients are initialized in TestMain.
	require.NotNil(t, workspaceClient, "workspace client should have been initialized")
	require.NotNil(t, k8sClient, "k8sClient should have been initialized")

	t.Run("should create a workspace resource via the gateway API", func(t *testing.T) {
		t.Parallel()

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
			key := client.ObjectKey{
				Namespace: "e2e-ecp", // All workspaces are created in the control plane namespace
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
	})

	t.Run("should delete a workspace resource via the gateway API", func(t *testing.T) {
		t.Parallel()

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
			key := client.ObjectKey{Namespace: "e2e-ecp", Name: workspaceName}
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
			key := client.ObjectKey{Namespace: "e2e-ecp", Name: workspaceName}
			if err := k8sClient.Get(ctx, key, &createdWorkspace); err != nil {
				// We expect it to be gone eventually, but checking for deletion timestamp first
				return false, nil
			}
			// The delegator marks it for deletion, it doesn't remove it immediately
			if createdWorkspace.GetDeletionTimestamp() != nil {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "workspace CR should have a deletion timestamp")
	})
}
