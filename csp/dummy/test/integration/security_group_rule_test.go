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
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

func TestSecurityGroupRule(t *testing.T) {
	t.Parallel()

	t.Run("should create a security group rule resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-sgr-create-" + uuid.New().String()[:8]
		ruleDomain := &securitygroupruledom.SecurityGroupRule{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
			},
			Spec: securitygroupruledom.SecurityGroupRuleSpec{
				Direction: "ingress",
				Protocol:  "tcp",
			},
		}

		_, err := securityGroupRuleRepo.Create(t.Context(), ruleDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := &securitygroupruledom.SecurityGroupRule{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
				},
			}
			if err := securityGroupRuleRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "security group rule resource should become active")
	})

	t.Run("should delete a security group rule resource", func(t *testing.T) {
		t.Parallel()

		resourceName := "test-sgr-delete-" + uuid.New().String()[:8]
		ruleDomain := &securitygroupruledom.SecurityGroupRule{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
				Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
			},
			Spec: securitygroupruledom.SecurityGroupRuleSpec{
				Direction: "egress",
				Protocol:  "udp",
			},
		}

		_, err := securityGroupRuleRepo.Create(t.Context(), ruleDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := &securitygroupruledom.SecurityGroupRule{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
				},
			}
			if err := securityGroupRuleRepo.Load(ctx, &loaded); err != nil {
				return false, err
			}
			return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
		})
		require.NoError(t, err, "security group rule resource should become active before deletion")

		err = securityGroupRuleRepo.Delete(t.Context(), ruleDomain)
		require.NoError(t, err)

		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loaded := &securitygroupruledom.SecurityGroupRule{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: resourceName},
					Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
				},
			}
			if err := securityGroupRuleRepo.Load(ctx, &loaded); err != nil {
				if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "security group rule resource should be deleted")
	})
}
