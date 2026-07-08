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

const (
	testTenant = "test-tenant"
)

func TestRoleAPI(t *testing.T) {
	//
	// Given the global gateway is running and the authorization client has been initialized.
	require.NotNil(t, authClient, "authorization client should have been initialized")

	t.Run("should create and retrieve a role", func(t *testing.T) {
		//
		// Given a unique role name.
		roleName := "e2e-role-" + uuid.New().String()[:8]
		roleBody := schema.Role{
			Spec: schema.RoleSpec{
				Permissions: []schema.Permission{
					{
						Provider:  "seca.compute",
						Resources: []string{"instances"},
						Verb:      []string{"get", "list"},
					},
				},
			},
		}

		//
		// When we create a role via the REST API.
		createResp, err := authClient.CreateOrUpdateRoleWithResponse(
			context.Background(),
			testTenant,
			roleName,
			&authv1.CreateOrUpdateRoleParams{},
			roleBody,
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode(), "expected HTTP 200 OK on role creation")
		require.NotNil(t, createResp.JSON200, "created role response body should not be nil")
		require.NotNil(t, createResp.JSON200.Metadata, "created role should have metadata")
		require.Equal(t, roleName, createResp.JSON200.Metadata.Name, "role name should match")

		//
		// And we can retrieve it back.
		getResp, err := authClient.GetRoleWithResponse(context.Background(), testTenant, roleName)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "expected HTTP 200 OK on role get")
		require.NotNil(t, getResp.JSON200, "get role response body should not be nil")
		require.NotNil(t, getResp.JSON200.Metadata)
		require.Equal(t, roleName, getResp.JSON200.Metadata.Name)
		require.Len(t, getResp.JSON200.Spec.Permissions, 1)
		require.Equal(t, "seca.compute", getResp.JSON200.Spec.Permissions[0].Provider)

		//
		// Cleanup: delete the role.
		deleteResp, err := authClient.DeleteRoleWithResponse(
			context.Background(),
			testTenant,
			roleName,
			&authv1.DeleteRoleParams{},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, deleteResp.StatusCode(), "expected HTTP 202 Accepted on role deletion")
	})

	t.Run("should list roles for a tenant", func(t *testing.T) {
		//
		// Given a unique role name.
		roleName := "e2e-list-role-" + uuid.New().String()[:8]
		roleBody := schema.Role{
			Spec: schema.RoleSpec{
				Permissions: []schema.Permission{
					{
						Provider:  "seca.authorization",
						Resources: []string{"roles"},
						Verb:      []string{"get"},
					},
				},
			},
		}

		//
		// Given a role exists.
		createResp, err := authClient.CreateOrUpdateRoleWithResponse(
			context.Background(),
			testTenant,
			roleName,
			&authv1.CreateOrUpdateRoleParams{},
			roleBody,
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, createResp.StatusCode())

		//
		// When we list roles.
		listResp, err := authClient.ListRolesWithResponse(context.Background(), testTenant, &authv1.ListRolesParams{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, listResp.StatusCode(), "expected HTTP 200 OK on list roles")
		require.NotNil(t, listResp.JSON200, "list roles response body should not be nil")
		require.NotNil(t, listResp.JSON200.Items, "role list items should not be nil")

		//
		// Then the created role should appear in the list.
		found := false
		for _, r := range listResp.JSON200.Items {
			if r.Metadata != nil && r.Metadata.Name == roleName {
				found = true
				break
			}
		}
		require.True(t, found, "should find the created role %q in the list", roleName)

		//
		// Cleanup: delete the role.
		deleteResp, err := authClient.DeleteRoleWithResponse(
			context.Background(),
			testTenant,
			roleName,
			&authv1.DeleteRoleParams{},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, deleteResp.StatusCode())
	})

	t.Run("should return 404 for a non-existent role", func(t *testing.T) {
		//
		// When we try to get a role that does not exist.
		resp, err := authClient.GetRoleWithResponse(context.Background(), testTenant, "non-existent-role-xyz")
		require.NoError(t, err)

		//
		// Then it should return 404.
		require.Equal(t, http.StatusNotFound, resp.StatusCode(), "expected HTTP 404 for non-existent role")
	})
}
