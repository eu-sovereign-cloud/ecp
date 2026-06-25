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
	t.Parallel()

	t.Run("should create a role resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-role-create-" + uuid.New().String()[:8]
		roleDomain := &roledom.Role{
			GlobalTenantMetadata: commondomain.GlobalTenantMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant"},
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

		_, err := roleRepo.Create(t.Context(), roleDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedRole := &roledom.Role{
				GlobalTenantMetadata: commondomain.GlobalTenantMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := roleRepo.Load(ctx, &loadedRole); err != nil {
				return false, err
			}
			return loadedRole.Status != nil && loadedRole.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "role resource should become active")
	})

	t.Run("should delete a role resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-role-delete-" + uuid.New().String()[:8]
		roleDomain := &roledom.Role{
			GlobalTenantMetadata: commondomain.GlobalTenantMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant"},
			},
			Spec: roledom.RoleSpec{
				Permissions: []roledom.Permission{
					{
						Provider:  "seca.authorization",
						Resources: []string{"roles"},
						Verb:      []string{"get"},
					},
				},
			},
		}

		_, err := roleRepo.Create(t.Context(), roleDomain)
		require.NoError(t, err)

		// Wait until active before deleting.
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := &roledom.Role{
				GlobalTenantMetadata: commondomain.GlobalTenantMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := roleRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "role should become active before deletion")

		// Delete the role.
		loadedRole := &roledom.Role{
			GlobalTenantMetadata: commondomain.GlobalTenantMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant"},
			},
		}
		if err := roleRepo.Load(t.Context(), &loadedRole); err != nil {
			t.Fatalf("failed to reload role: %v", err)
		}
		_, err = roleRepo.Delete(t.Context(), loadedRole)
		require.NoError(t, err)
	})
}
