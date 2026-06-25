//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role-assignment/v1"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

func TestRoleAssignment(t *testing.T) {
	t.Parallel()

	t.Run("should create a role assignment resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-ra-create-" + uuid.New().String()[:8]
		raDomain := &radom.RoleAssignment{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant"},
			},
			Spec: radom.RoleAssignmentSpec{
				Subs:   []string{"user1@example.com"},
				Scopes: []radom.RoleAssignmentScope{{Tenants: []string{"test-tenant"}}},
				Roles:  []string{"workspace-viewer"},
			},
		}

		_, err := roleAssignmentRepo.Create(t.Context(), raDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedRA := &radom.RoleAssignment{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := roleAssignmentRepo.Load(ctx, &loadedRA); err != nil {
				return false, err
			}
			return loadedRA.Status != nil && loadedRA.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "role assignment resource should become active")
	})

	t.Run("should delete a role assignment resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-ra-delete-" + uuid.New().String()[:8]
		raDomain := &radom.RoleAssignment{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: "test-tenant"},
			},
			Spec: radom.RoleAssignmentSpec{
				Subs:   []string{"user1@example.com"},
				Scopes: []radom.RoleAssignmentScope{{Tenants: []string{"test-tenant"}}},
				Roles:  []string{"workspace-viewer"},
			},
		}
		_, err := roleAssignmentRepo.Create(t.Context(), raDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedRA := &radom.RoleAssignment{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := roleAssignmentRepo.Load(ctx, &loadedRA); err != nil {
				return false, err
			}
			return loadedRA.Status != nil && loadedRA.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "role assignment resource should become active before deletion")

		err = roleAssignmentRepo.Delete(t.Context(), raDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedRA := &radom.RoleAssignment{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: "test-tenant"},
				},
			}
			if err := roleAssignmentRepo.Load(ctx, &loadedRA); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "role assignment resource should be deleted")
	})
}
