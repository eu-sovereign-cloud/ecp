//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// TestWorkspaceAPI exercises the regional gateway's workspace REST handler. Create,
// read-back, and delete are verified through the gateway's HTTP responses;
// reconciliation to Active is covered by the delegator suite, so this suite needs no
// reconciler.
func TestWorkspaceAPI(t *testing.T) {
	t.Run("should create and retrieve a workspace resource via the gateway API", func(t *testing.T) {
		//
		// Given a unique workspace name
		workspaceName := "test-ws-create-" + uuid.New().String()[:8]

		//
		// When we create it through the gateway
		createResp, err := workspaceClient.CreateOrUpdateWorkspaceWithResponse(context.Background(), testTenant, workspaceName, nil, schema.Workspace{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())

		//
		// Then it can be read back
		getResp, err := workspaceClient.GetWorkspaceWithResponse(context.Background(), testTenant, workspaceName)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode())
		require.NotNil(t, getResp.JSON200)
		require.NotNil(t, getResp.JSON200.Metadata)
		require.Equal(t, workspaceName, getResp.JSON200.Metadata.Name)

		//
		// And it can be deleted
		delResp, err := workspaceClient.DeleteWorkspaceWithResponse(context.Background(), testTenant, workspaceName, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, delResp.StatusCode())
	})

	t.Run("should delete a workspace resource via the gateway API", func(t *testing.T) {
		//
		// Given a workspace that has been created
		workspaceName := "test-ws-delete-" + uuid.New().String()[:8]
		createResp, err := workspaceClient.CreateOrUpdateWorkspaceWithResponse(context.Background(), testTenant, workspaceName, nil, schema.Workspace{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())

		//
		// When we delete it, the gateway accepts the request
		delResp, err := workspaceClient.DeleteWorkspaceWithResponse(context.Background(), testTenant, workspaceName, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, delResp.StatusCode())
	})
}
