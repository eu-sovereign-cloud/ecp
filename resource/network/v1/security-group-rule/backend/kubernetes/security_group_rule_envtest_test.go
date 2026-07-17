//go:build envtest

package kubernetes_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/dynamic"
	k8sinterface "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	"github.com/eu-sovereign-cloud/ecp/resource/common/frontend/testutil"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
)

func TestSecurityGroupRuleBackend_CreateAndGetSecurityGroupRule(t *testing.T) {
	t.Parallel()

	// Use a config copy with higher rate limits to avoid rate limiter exhaustion
	// during the adapter's status polling loop.
	testCfg := rest.CopyConfig(cfg)
	testCfg.QPS = 50
	testCfg.Burst = 100

	dynClient, err := dynamic.NewForConfig(testCfg)
	require.NoError(t, err)

	clientset, err := k8sinterface.NewForConfig(testCfg)
	require.NoError(t, err)

	// Create valid Kubernetes namespace name (lowercase, alphanumeric and hyphens only).
	// Keep tenant short to fit within 63 char label limit.
	tenant := "t-sgr-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	const (
		workspace = "test-workspace"
		ruleName  = "test-sg-rule"
	)

	// SecurityGroupRule CRs live in the namespace computed from tenant + workspace.
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant, Workspace: workspace})

	ctx := context.Background()

	// Create the namespace before creating SecurityGroupRule resources. The WriterAdapter
	// does not manage namespaces automatically, so it must exist in advance.
	_, err = k8sadapter.CreateNamespace(ctx, clientset, namespace, map[string]string{
		k8slabels.InternalTenantLabel: tenant,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset, namespace)
	})

	writerRepo := k8sadapter.NewWriterAdapter[*securitygroupruledom.SecurityGroupRule](
		dynClient,
		SecurityGroupRuleGVR,
		slog.Default(),
		SecurityGroupRuleToCR,
		SecurityGroupRuleFromCR,
	)

	readerRepo := k8sadapter.NewReaderAdapter[*securitygroupruledom.SecurityGroupRule](
		dynClient,
		SecurityGroupRuleGVR,
		slog.Default(),
		SecurityGroupRuleFromCR,
	)

	newRuleDomain := func() *securitygroupruledom.SecurityGroupRule {
		return &securitygroupruledom.SecurityGroupRule{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: ruleName},
				Scope:          kernelresource.Scope{Tenant: tenant, Workspace: workspace},
				Labels:         map[string]string{k8slabels.InternalTenantLabel: tenant},
			},
			Spec: securitygroupruledom.SecurityGroupRuleSpec{
				Direction: "ingress",
				Protocol:  "tcp",
				Ports:     &securitygroupruledom.Ports{From: 80, To: 443},
			},
		}
	}

	t.Run("create_update_list_delete_security_group_rule", func(t *testing.T) {
		createDomain := newRuleDomain()

		// Simulate a controller that sets status.state after the CR is created.
		statusCfg := rest.CopyConfig(cfg)
		statusCfg.QPS = 50
		statusCfg.Burst = 100
		statusClient, err := dynamic.NewForConfig(statusCfg)
		require.NoError(t, err)

		statusCtx, statusCancel := context.WithCancel(ctx)
		defer statusCancel()
		go testutil.SimulateStatusController(statusCtx, statusClient, SecurityGroupRuleGVR, namespace, ruleName, map[string]interface{}{
			"state": "active",
		})

		// Create the SecurityGroupRule.
		result, err := writerRepo.Create(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, result)
		created := *result
		require.Equal(t, ruleName, created.Name)
		require.Equal(t, "ingress", created.Spec.Direction)
		require.Equal(t, "tcp", created.Spec.Protocol)
		require.Equal(t, 80, created.Spec.Ports.From)

		// Get the SecurityGroupRule and verify it matches.
		rule := &securitygroupruledom.SecurityGroupRule{}
		rule.Name = ruleName
		rule.Tenant = tenant
		rule.Workspace = workspace
		err = readerRepo.Load(ctx, &rule)
		require.NoError(t, err)
		require.NotNil(t, rule)
		require.Equal(t, "ingress", rule.Spec.Direction)

		// Update the spec (protocol change).
		updateDomain := newRuleDomain()
		updateDomain.ResourceVersion = created.ResourceVersion
		updateDomain.Spec.Protocol = "udp"

		updateResult, err := writerRepo.Update(ctx, updateDomain)
		require.NoError(t, err)
		require.NotNil(t, updateResult)
		require.Equal(t, "udp", (*updateResult).Spec.Protocol)

		// Verify update with a Get.
		rule2 := &securitygroupruledom.SecurityGroupRule{}
		rule2.Name = ruleName
		rule2.Tenant = tenant
		rule2.Workspace = workspace
		err = readerRepo.Load(ctx, &rule2)
		require.NoError(t, err)
		require.Equal(t, "udp", rule2.Spec.Protocol)

		// List SecurityGroupRules and verify ours exists.
		var items []*securitygroupruledom.SecurityGroupRule
		_, err = readerRepo.List(ctx, kernelresource.ListParams{Scope: kernelresource.Scope{Tenant: tenant, Workspace: workspace}}, &items)
		require.NoError(t, err)
		require.NotEmpty(t, items)
		found := false
		for _, it := range items {
			if it != nil && it.Name == ruleName {
				found = true
				break
			}
		}
		require.True(t, found, "expected security group rule to be present in list")

		// Delete the SecurityGroupRule.
		del := &securitygroupruledom.SecurityGroupRule{}
		del.Name = ruleName
		del.Tenant = tenant
		del.Workspace = workspace
		err = writerRepo.Delete(ctx, del)
		require.NoError(t, err)
	})

	t.Run("get_nonexistent_security_group_rule", func(t *testing.T) {
		rule := &securitygroupruledom.SecurityGroupRule{}
		rule.Name = "missing-security-group-rule"
		rule.Tenant = tenant
		rule.Workspace = workspace
		err := readerRepo.Load(ctx, &rule)
		require.Error(t, err)
		domainErr := kernel.AsError(err)
		require.NotNil(t, domainErr)
		require.Equal(t, kernel.KindNotFound, domainErr.Kind)
	})
}
