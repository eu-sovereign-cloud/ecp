//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
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

		body, err := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		require.NoError(t, err)

		var createdRole schema.Role
		require.NoError(t, json.Unmarshal(body, &createdRole), "should parse created role response")
		require.NotNil(t, createdRole.Metadata, "created role should have metadata")
		require.Equal(t, roleName, createdRole.Metadata.Name, "role name should match")

		//
		// And we can retrieve it back.
		getResp, err := authClient.GetRoleWithResponse(context.Background(), testTenant, roleName)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "expected HTTP 200 OK on role get")

		getBody, err := io.ReadAll(getResp.Body)
		_ = getResp.Body.Close()
		require.NoError(t, err)

		var fetchedRole schema.Role
		require.NoError(t, json.Unmarshal(getBody, &fetchedRole), "should parse fetched role response")
		require.NotNil(t, fetchedRole.Metadata)
		require.Equal(t, roleName, fetchedRole.Metadata.Name)
		require.Len(t, fetchedRole.Spec.Permissions, 1)
		require.Equal(t, "seca.compute", fetchedRole.Spec.Permissions[0].Provider)

		//
		// Cleanup: delete the role.
		deleteResp, err := authClient.DeleteRoleWithResponse(
			context.Background(),
			testTenant,
			roleName,
			&authv1.DeleteRoleParams{},
		)
		require.NoError(t, err)
		_ = deleteResp.Body.Close()
		require.Equal(t, http.StatusOK, deleteResp.StatusCode(), "expected HTTP 200 OK on role deletion")
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
		_ = createResp.Body.Close()
		require.Equal(t, http.StatusOK, createResp.StatusCode())

		//
		// When we list roles.
		listResp, err := authClient.ListRolesWithResponse(context.Background(), testTenant, &authv1.ListRolesParams{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, listResp.StatusCode(), "expected HTTP 200 OK on list roles")

		listBody, err := io.ReadAll(listResp.Body)
		_ = listResp.Body.Close()
		require.NoError(t, err)

		var roleIterator authv1.RoleIterator
		require.NoError(t, json.Unmarshal(listBody, &roleIterator), "should parse role list response")
		require.NotNil(t, roleIterator.Items, "role list items should not be nil")

		//
		// Then the created role should appear in the list.
		found := false
		for _, r := range roleIterator.Items {
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
		_ = deleteResp.Body.Close()
		require.Equal(t, http.StatusOK, deleteResp.StatusCode())
	})

	t.Run("should return 404 for a non-existent role", func(t *testing.T) {
		//
		// When we try to get a role that does not exist.
		resp, err := authClient.GetRoleWithResponse(context.Background(), testTenant, "non-existent-role-xyz")
		require.NoError(t, err)
		_ = resp.Body.Close()

		//
		// Then it should return 404.
		require.Equal(t, http.StatusNotFound, resp.StatusCode(), "expected HTTP 404 for non-existent role")
	})
}
