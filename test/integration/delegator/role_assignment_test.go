//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

func TestRoleAssignment(t *testing.T) {
	t.Run("should create a role assignment resource and reconcile it to active", func(t *testing.T) {
		//
		// Given a unique role assignment domain resource definition
		resourceName := "test-role-assignment-create-" + uuid.New().String()[:8]
		raDomain := &radom.RoleAssignment{
			GlobalTenantMetadata: commondomain.GlobalTenantMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name: resourceName,
				},
				Scope: kernelresource.Scope{
					Tenant: testTenant,
				},
			},
			Spec: radom.RoleAssignmentSpec{
				Subs:   []string{"user1@example.com"},
				Scopes: []radom.RoleAssignmentScope{{Tenants: []string{testTenant}}},
				Roles:  []string{"workspace-viewer"},
			},
		}

		//
		// When we create the role assignment resource via the adapter
		_, err := roleAssignmentRepo.Create(t.Context(), raDomain)
		require.NoError(t, err)

		//
		// Then the resource should eventually become active
		var loadedRA *radom.RoleAssignment

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedRA = &radom.RoleAssignment{
				GlobalTenantMetadata: commondomain.GlobalTenantMetadata{
					CommonMetadata: commondomain.CommonMetadata{
						Name: resourceName,
					},
					Scope: kernelresource.Scope{
						Tenant: testTenant,
					},
				},
			}
			if err := roleAssignmentRepo.Load(ctx, &loadedRA); err != nil {
				return false, err
			}
			if loadedRA.Status != nil && loadedRA.Status.State == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "role assignment resource should become active")
		require.NotNil(t, loadedRA)
		require.NotNil(t, loadedRA.Status)
		require.Equal(t, commondomain.ResourceStateActive, loadedRA.Status.State)

		//
		// And we can cleanup the role assignment
		err = roleAssignmentRepo.Delete(t.Context(), raDomain)
		require.NoError(t, err)
	})
}
