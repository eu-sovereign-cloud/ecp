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
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
)

func TestSecurityGroupBackend_CreateAndGetSecurityGroup(t *testing.T) {
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
	tenant := "t-sg-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	const (
		workspace = "test-workspace"
		sgName    = "test-security-group"
	)

	// SecurityGroup CRs live in the namespace computed from tenant + workspace.
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant, Workspace: workspace})

	ctx := context.Background()

	// Create the namespace before creating SecurityGroup resources. The WriterAdapter
	// does not manage namespaces automatically, so it must exist in advance.
	err = k8sadapter.CreateNamespace(ctx, clientset, namespace, map[string]string{
		k8slabels.InternalTenantLabel: tenant,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset, namespace)
	})

	writerRepo := k8sadapter.NewWriterAdapter[*securitygroupdom.SecurityGroup](
		dynClient,
		SecurityGroupGVR,
		slog.Default(),
		SecurityGroupToCR,
		SecurityGroupFromCR,
	)

	readerRepo := k8sadapter.NewReaderAdapter[*securitygroupdom.SecurityGroup](
		dynClient,
		SecurityGroupGVR,
		slog.Default(),
		SecurityGroupFromCR,
	)

	newSGDomain := func() *securitygroupdom.SecurityGroup {
		return &securitygroupdom.SecurityGroup{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: sgName},
				Scope:          kernelresource.Scope{Tenant: tenant, Workspace: workspace},
				Labels:         map[string]string{k8slabels.InternalTenantLabel: tenant},
			},
			Spec: securitygroupdom.SecurityGroupSpec{
				Rules: []securitygroupdom.SecurityGroupRuleSpec{
					{Direction: "ingress", Protocol: "tcp", Ports: &securitygroupdom.Ports{From: 80, To: 443}},
				},
			},
		}
	}

	t.Run("create_update_list_delete_security_group", func(t *testing.T) {
		createDomain := newSGDomain()

		// Simulate a controller that sets status.state after the CR is created.
		statusCfg := rest.CopyConfig(cfg)
		statusCfg.QPS = 50
		statusCfg.Burst = 100
		statusClient, err := dynamic.NewForConfig(statusCfg)
		require.NoError(t, err)

		statusCtx, statusCancel := context.WithCancel(ctx)
		defer statusCancel()
		go testutil.SimulateStatusController(statusCtx, statusClient, SecurityGroupGVR, namespace, sgName, map[string]interface{}{
			"state": "active",
		})

		// Create the SecurityGroup.
		result, err := writerRepo.Create(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, result)
		created := *result
		require.Equal(t, sgName, created.Name)
		require.Len(t, created.Spec.Rules, 1)
		require.Equal(t, "ingress", created.Spec.Rules[0].Direction)

		// Get the SecurityGroup and verify it matches.
		sg := &securitygroupdom.SecurityGroup{}
		sg.Name = sgName
		sg.Tenant = tenant
		sg.Workspace = workspace
		err = readerRepo.Load(ctx, &sg)
		require.NoError(t, err)
		require.NotNil(t, sg)
		require.Len(t, sg.Spec.Rules, 1)

		// Update the spec (add a rule ref).
		updateDomain := newSGDomain()
		updateDomain.ResourceVersion = created.ResourceVersion
		// Reference.resource: {collection}/{name}
		// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
		updateDomain.Spec.RuleRefs = []commondomain.Reference{{Resource: "security-group-rules/shared-rule"}}

		updateResult, err := writerRepo.Update(ctx, updateDomain)
		require.NoError(t, err)
		require.NotNil(t, updateResult)
		require.Len(t, (*updateResult).Spec.RuleRefs, 1)
		require.Equal(t, "security-group-rules/shared-rule", (*updateResult).Spec.RuleRefs[0].Resource)

		// Verify update with a Get.
		sg2 := &securitygroupdom.SecurityGroup{}
		sg2.Name = sgName
		sg2.Tenant = tenant
		sg2.Workspace = workspace
		err = readerRepo.Load(ctx, &sg2)
		require.NoError(t, err)
		require.Len(t, sg2.Spec.RuleRefs, 1)

		// List SecurityGroups and verify ours exists.
		var items []*securitygroupdom.SecurityGroup
		_, err = readerRepo.List(ctx, kernelresource.ListParams{Scope: kernelresource.Scope{Tenant: tenant, Workspace: workspace}}, &items)
		require.NoError(t, err)
		require.NotEmpty(t, items)
		found := false
		for _, it := range items {
			if it != nil && it.Name == sgName {
				found = true
				break
			}
		}
		require.True(t, found, "expected security group to be present in list")

		// Delete the SecurityGroup.
		del := &securitygroupdom.SecurityGroup{}
		del.Name = sgName
		del.Tenant = tenant
		del.Workspace = workspace
		err = writerRepo.Delete(ctx, del)
		require.NoError(t, err)
	})

	t.Run("get_nonexistent_security_group", func(t *testing.T) {
		sg := &securitygroupdom.SecurityGroup{}
		sg.Name = "missing-security-group"
		sg.Tenant = tenant
		sg.Workspace = workspace
		err := readerRepo.Load(ctx, &sg)
		require.Error(t, err)
		domainErr := kernel.AsError(err)
		require.NotNil(t, domainErr)
		require.Equal(t, kernel.KindNotFound, domainErr.Kind)
	})
}
