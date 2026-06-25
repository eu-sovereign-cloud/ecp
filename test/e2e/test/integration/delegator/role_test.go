//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role/v1"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

func TestRole(t *testing.T) {
	t.Run("should create a role resource and reconcile it to active", func(t *testing.T) {
		//
		// Given a unique role domain resource definition
		resourceName := "test-role-create-" + uuid.New().String()[:8]
		roleDomain := &roledom.Role{
			GlobalTenantMetadata: commondomain.GlobalTenantMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name: resourceName,
				},
				Scope: kernelresource.Scope{
					Tenant: testTenant,
				},
			},
			Spec: roledom.RoleSpec{
				Permissions: []roledom.Permission{
					{
						Provider:  "seca.compute",
						Resources: []string{"instances"},
						Verb:      []string{"get", "list"},
					},
				},
			},
		}

		//
		// When we create the role resource via the adapter
		_, err := roleRepo.Create(t.Context(), roleDomain)
		require.NoError(t, err)

		//
		// Then the resource should eventually become active
		var loadedRole *roledom.Role

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedRole = &roledom.Role{
				GlobalTenantMetadata: commondomain.GlobalTenantMetadata{
					CommonMetadata: commondomain.CommonMetadata{
						Name: resourceName,
					},
					Scope: kernelresource.Scope{
						Tenant: testTenant,
					},
				},
			}
			if err := roleRepo.Load(ctx, &loadedRole); err != nil {
				return false, err
			}
			if loadedRole.Status != nil && loadedRole.Status.State == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "role resource should become active")
		require.NotNil(t, loadedRole)
		require.NotNil(t, loadedRole.Status)
		require.Equal(t, commondomain.ResourceStateActive, loadedRole.Status.State)

		//
		// And we can cleanup the role
		err = roleRepo.Delete(t.Context(), roleDomain)
		require.NoError(t, err)
	})
}
