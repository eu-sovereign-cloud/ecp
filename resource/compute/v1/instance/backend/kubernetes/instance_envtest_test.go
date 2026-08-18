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
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	. "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
)

func TestInstanceBackend_CreateAndGetInstance(t *testing.T) {
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
	tenant := "t-inst-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	const (
		workspace    = "test-workspace"
		instanceName = "test-instance"
	)

	// Instance CRs live in the namespace computed from tenant + workspace.
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant, Workspace: workspace})

	ctx := context.Background()

	// Create the namespace before creating Instance resources. The WriterAdapter
	// does not manage namespaces automatically, so it must exist in advance.
	err = k8sadapter.CreateNamespace(ctx, clientset, namespace, map[string]string{
		k8slabels.InternalTenantLabel: tenant,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset, namespace)
	})

	writerRepo := k8sadapter.NewWriterAdapter[*instancedom.Instance](
		dynClient,
		InstanceGVR,
		slog.Default(),
		InstanceToCR,
		InstanceFromCR,
	)

	readerRepo := k8sadapter.NewReaderAdapter[*instancedom.Instance](
		dynClient,
		InstanceGVR,
		slog.Default(),
		InstanceFromCR,
	)

	newInstanceDomain := func() *instancedom.Instance {
		return &instancedom.Instance{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: instanceName},
				Scope:          kernelresource.Scope{Tenant: tenant, Workspace: workspace},
				Labels:         map[string]string{k8slabels.InternalTenantLabel: tenant},
			},
			Spec: instancedom.InstanceSpec{
				// Reference.resource: {collection}/{name}
				// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
				BootVolume: instancedom.VolumeReference{DeviceRef: commondomain.Reference{Resource: "block-storages/boot"}},
				SkuRef:     commondomain.Reference{Resource: "skus/standard-instance"},
				Zone:       "zone-1",
			},
		}
	}

	t.Run("create_update_list_delete_instance", func(t *testing.T) {
		createDomain := newInstanceDomain()

		// Simulate a controller that sets status.state after the CR is created.
		statusCfg := rest.CopyConfig(cfg)
		statusCfg.QPS = 50
		statusCfg.Burst = 100
		statusClient, err := dynamic.NewForConfig(statusCfg)
		require.NoError(t, err)

		statusCtx, statusCancel := context.WithCancel(ctx)
		defer statusCancel()
		go testutil.SimulateStatusController(statusCtx, statusClient, InstanceGVR, namespace, instanceName, map[string]interface{}{
			"state": "active",
		})

		// Create the Instance.
		result, err := writerRepo.Create(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, result)
		created := *result
		require.Equal(t, instanceName, created.Name)
		require.Equal(t, "zone-1", created.Spec.Zone)
		require.Equal(t, "skus/standard-instance", created.Spec.SkuRef.Resource)

		// Get the Instance and verify it matches.
		inst := &instancedom.Instance{}
		inst.Name = instanceName
		inst.Tenant = tenant
		inst.Workspace = workspace
		err = readerRepo.Load(ctx, &inst)
		require.NoError(t, err)
		require.NotNil(t, inst)
		require.Equal(t, "zone-1", inst.Spec.Zone)

		// Update the spec (mutable field: anti-affinity group).
		inst2 := &instancedom.Instance{}
		inst2.Name = instanceName
		inst2.Tenant = tenant
		inst2.Workspace = workspace
		err = readerRepo.Load(ctx, &inst2)
		require.NoError(t, err)

		updateDomain := newInstanceDomain()
		updateDomain.ResourceVersion = inst2.ResourceVersion
		updateDomain.Spec.AntiAffinityGroup = "group-a"

		updateResult, err := writerRepo.Update(ctx, updateDomain)
		require.NoError(t, err)
		require.NotNil(t, updateResult)
		require.Equal(t, "group-a", (*updateResult).Spec.AntiAffinityGroup)

		// Verify update with a Get.
		inst3 := &instancedom.Instance{}
		inst3.Name = instanceName
		inst3.Tenant = tenant
		inst3.Workspace = workspace
		err = readerRepo.Load(ctx, &inst3)
		require.NoError(t, err)
		require.Equal(t, "group-a", inst3.Spec.AntiAffinityGroup)

		// List Instances and verify ours exists.
		var items []*instancedom.Instance
		_, err = readerRepo.List(ctx, kernelresource.ListParams{Scope: kernelresource.Scope{Tenant: tenant, Workspace: workspace}}, &items)
		require.NoError(t, err)
		require.NotEmpty(t, items)
		found := false
		for _, it := range items {
			if it != nil && it.Name == instanceName {
				found = true
				break
			}
		}
		require.True(t, found, "expected instance to be present in list")

		// Delete the Instance.
		del := &instancedom.Instance{}
		del.Name = instanceName
		del.Tenant = tenant
		del.Workspace = workspace
		err = writerRepo.Delete(ctx, del)
		require.NoError(t, err)
	})

	t.Run("get_nonexistent_instance", func(t *testing.T) {
		inst := &instancedom.Instance{}
		inst.Name = "missing-instance"
		inst.Tenant = tenant
		inst.Workspace = workspace
		err := readerRepo.Load(ctx, &inst)
		require.Error(t, err)
		domainErr := kernel.AsError(err)
		require.NotNil(t, domainErr)
		require.Equal(t, kernel.KindNotFound, domainErr.Kind)
	})

	t.Run("reject_skuref_mutation", func(t *testing.T) {
		const immutableInstanceName = "test-instance-immutable"

		createDomain := &instancedom.Instance{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: immutableInstanceName},
				Scope:          kernelresource.Scope{Tenant: tenant, Workspace: workspace},
				Labels:         map[string]string{k8slabels.InternalTenantLabel: tenant},
			},
			Spec: instancedom.InstanceSpec{
				BootVolume: instancedom.VolumeReference{DeviceRef: commondomain.Reference{Resource: "block-storages/boot"}},
				SkuRef:     commondomain.Reference{Resource: "skus/sku-original"},
				Zone:       "zone-1",
			},
		}

		statusClient, err := dynamic.NewForConfig(testCfg)
		require.NoError(t, err)
		statusCtx, statusCancel := context.WithCancel(ctx)
		defer statusCancel()
		go testutil.SimulateStatusController(statusCtx, statusClient, InstanceGVR, namespace, immutableInstanceName, map[string]interface{}{
			"state": "active",
		})

		_, err = writerRepo.Create(ctx, createDomain)
		require.NoError(t, err)

		current := &instancedom.Instance{}
		current.Name = immutableInstanceName
		current.Tenant = tenant
		current.Workspace = workspace
		err = readerRepo.Load(ctx, &current)
		require.NoError(t, err)

		// Attempt to change the immutable skuRef — the apiserver CEL rule must reject it.
		mutated := &instancedom.Instance{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name:            immutableInstanceName,
					ResourceVersion: current.ResourceVersion,
				},
				Scope: kernelresource.Scope{Tenant: tenant, Workspace: workspace},
			},
			Spec: instancedom.InstanceSpec{
				BootVolume: instancedom.VolumeReference{DeviceRef: commondomain.Reference{Resource: "block-storages/boot"}},
				SkuRef:     commondomain.Reference{Resource: "skus/sku-changed"},
				Zone:       "zone-1",
			},
		}
		_, err = writerRepo.Update(ctx, mutated)
		require.Error(t, err, "changing the immutable skuRef should be rejected")
		require.ErrorContains(t, err, "spec.skuRef is immutable")
	})
}
