//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	authv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

func TestRoleAssignmentAPI(t *testing.T) {
	//
	// Given the global gateway is running and the authorization client has been initialized.
	require.NotNil(t, authClient, "authorization client should have been initialized")

	t.Run("should create and retrieve a role assignment", func(t *testing.T) {
		//
		// Given a unique role assignment name.
		raName := "e2e-ra-" + uuid.New().String()[:8]
		raBody := schema.RoleAssignment{
			Spec: schema.RoleAssignmentSpec{
				Subs:   []string{"user1@example.com"},
				Scopes: []schema.RoleAssignmentScope{{Tenants: []string{testTenant}}},
				Roles:  []string{"workspace-viewer"},
			},
		}

		//
		// When we create a role assignment via the REST API.
		createResp, err := authClient.CreateOrUpdateRoleAssignmentWithResponse(
			context.Background(),
			testTenant,
			raName,
			&authv1.CreateOrUpdateRoleAssignmentParams{},
			raBody,
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode(), "expected HTTP 200 OK on role assignment creation")
		require.NotNil(t, createResp.JSON200, "created role assignment response body should not be nil")
		require.NotNil(t, createResp.JSON200.Metadata, "created role assignment should have metadata")
		require.Equal(t, raName, createResp.JSON200.Metadata.Name, "role assignment name should match")

		//
		// And we can retrieve it back.
		getResp, err := authClient.GetRoleAssignmentWithResponse(context.Background(), testTenant, raName)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "expected HTTP 200 OK on role assignment get")
		require.NotNil(t, getResp.JSON200, "get role assignment response body should not be nil")
		require.NotNil(t, getResp.JSON200.Metadata)
		require.Equal(t, raName, getResp.JSON200.Metadata.Name)
		require.Equal(t, []string{"workspace-viewer"}, getResp.JSON200.Spec.Roles)
		require.Equal(t, []string{"user1@example.com"}, getResp.JSON200.Spec.Subs)

		//
		// Cleanup: delete the role assignment.
		deleteResp, err := authClient.DeleteRoleAssignmentWithResponse(
			context.Background(),
			testTenant,
			raName,
			&authv1.DeleteRoleAssignmentParams{},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, deleteResp.StatusCode(), "expected HTTP 202 Accepted on role assignment deletion")
	})

	t.Run("should list role assignments for a tenant", func(t *testing.T) {
		//
		// Given a unique role assignment name.
		raName := "e2e-list-ra-" + uuid.New().String()[:8]
		raBody := schema.RoleAssignment{
			Spec: schema.RoleAssignmentSpec{
				Subs:   []string{"service-account-1"},
				Scopes: []schema.RoleAssignmentScope{{Tenants: []string{testTenant}}},
				Roles:  []string{"workspace-editor"},
			},
		}

		//
		// Given a role assignment exists.
		createResp, err := authClient.CreateOrUpdateRoleAssignmentWithResponse(
			context.Background(),
			testTenant,
			raName,
			&authv1.CreateOrUpdateRoleAssignmentParams{},
			raBody,
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())

		//
		// When we list role assignments.
		listResp, err := authClient.ListRoleAssignmentsWithResponse(context.Background(), testTenant, &authv1.ListRoleAssignmentsParams{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, listResp.StatusCode(), "expected HTTP 200 OK on list role assignments")
		require.NotNil(t, listResp.JSON200, "list role assignments response body should not be nil")
		require.NotNil(t, listResp.JSON200.Items, "role assignment list items should not be nil")

		//
		// Then the created role assignment should appear in the list.
		found := false
		for _, ra := range listResp.JSON200.Items {
			if ra.Metadata != nil && ra.Metadata.Name == raName {
				found = true
				break
			}
		}
		require.True(t, found, "should find the created role assignment %q in the list", raName)

		//
		// Cleanup: delete the role assignment.
		deleteResp, err := authClient.DeleteRoleAssignmentWithResponse(
			context.Background(),
			testTenant,
			raName,
			&authv1.DeleteRoleAssignmentParams{},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, deleteResp.StatusCode())
	})

	t.Run("should return 404 for a non-existent role assignment", func(t *testing.T) {
		//
		// When we try to get a role assignment that does not exist.
		resp, err := authClient.GetRoleAssignmentWithResponse(context.Background(), testTenant, "non-existent-role-assignment-xyz")
		require.NoError(t, err)

		//
		// Then it should return 404.
		require.Equal(t, http.StatusNotFound, resp.StatusCode(), "expected HTTP 404 for non-existent role assignment")
	})
}
