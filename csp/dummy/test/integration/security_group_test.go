//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

func TestSecurityGroup(t *testing.T) {
	t.Parallel()

	t.Run("should create a security group resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-sg-create-" + uuid.New().String()[:8]
		sgDomain := &securitygroupdom.SecurityGroup{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
			},
			Spec: securitygroupdom.SecurityGroupSpec{
				Rules: []securitygroupdom.SecurityGroupRuleSpec{
					{Direction: "ingress", Protocol: "tcp"},
				},
			},
		}

		_, err := securityGroupRepo.Create(t.Context(), sgDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := &securitygroupdom.SecurityGroup{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
				},
			}
			if err := securityGroupRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "security group resource should become active")
	})

	t.Run("should delete a security group resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-sg-delete-" + uuid.New().String()[:8]
		sgDomain := &securitygroupdom.SecurityGroup{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
			},
		}

		_, err := securityGroupRepo.Create(t.Context(), sgDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := &securitygroupdom.SecurityGroup{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
				},
			}
			if err := securityGroupRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "security group resource should become active before deletion")

		err = securityGroupRepo.Delete(t.Context(), sgDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := &securitygroupdom.SecurityGroup{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
				},
			}
			if err := securityGroupRepo.Load(ctx, &loaded); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "security group resource should be deleted")
	})
}
