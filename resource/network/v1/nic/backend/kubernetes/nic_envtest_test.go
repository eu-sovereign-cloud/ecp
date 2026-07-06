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
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
)

func TestNicBackend_CreateAndGetNic(t *testing.T) {
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
	tenant := "t-nic-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	const (
		workspace = "test-workspace"
		nicName   = "test-nic"
	)

	// NIC CRs live in the namespace computed from tenant + workspace.
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant, Workspace: workspace})

	ctx := context.Background()

	// Create the namespace before creating NIC resources. The WriterAdapter
	// does not manage namespaces automatically, so it must exist in advance.
	_, err = k8sadapter.CreateNamespace(ctx, clientset, namespace, map[string]string{
		k8slabels.InternalTenantLabel: tenant,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset, namespace)
	})

	writerRepo := k8sadapter.NewWriterAdapter[*nicdom.Nic](
		dynClient,
		NICGVR,
		slog.Default(),
		NicToCR,
		NicFromCR,
	)

	readerRepo := k8sadapter.NewReaderAdapter[*nicdom.Nic](
		dynClient,
		NICGVR,
		slog.Default(),
		NicFromCR,
	)

	newNicDomain := func() *nicdom.Nic {
		return &nicdom.Nic{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: nicName},
				Scope:          kernelresource.Scope{Tenant: tenant, Workspace: workspace},
				Labels:         map[string]string{k8slabels.InternalTenantLabel: tenant},
			},
			Spec: nicdom.NicSpec{
				Addresses: []string{"10.0.0.5"},
				SubnetRef: commondomain.Reference{Resource: "subnet-1"},
				SkuRef:    commondomain.Reference{Resource: "standard-nic"},
			},
		}
	}

	t.Run("create_update_list_delete_nic", func(t *testing.T) {
		createDomain := newNicDomain()

		// Simulate a controller that sets status.state after the CR is created.
		statusCfg := rest.CopyConfig(cfg)
		statusCfg.QPS = 50
		statusCfg.Burst = 100
		statusClient, err := dynamic.NewForConfig(statusCfg)
		require.NoError(t, err)

		statusCtx, statusCancel := context.WithCancel(ctx)
		defer statusCancel()
		go testutil.SimulateStatusController(statusCtx, statusClient, NICGVR, namespace, nicName, map[string]interface{}{
			"state": "active",
		})

		// Create the NIC
		result, err := writerRepo.Create(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, result)
		createdDomain := *result
		require.Equal(t, nicName, createdDomain.Name)
		require.Equal(t, []string{"10.0.0.5"}, createdDomain.Spec.Addresses)
		require.Equal(t, "subnet-1", createdDomain.Spec.SubnetRef.Resource)
		require.Equal(t, "standard-nic", createdDomain.Spec.SkuRef.Resource)

		// Get the NIC and verify it matches
		nic := &nicdom.Nic{}
		nic.Name = nicName
		nic.Tenant = tenant
		nic.Workspace = workspace
		err = readerRepo.Load(ctx, &nic)
		require.NoError(t, err)
		require.NotNil(t, nic)
		require.Equal(t, nicName, nic.Name)
		require.Equal(t, "subnet-1", nic.Spec.SubnetRef.Resource)
		require.Equal(t, "standard-nic", nic.Spec.SkuRef.Resource)

		// Get current resource version for the update
		nic2 := &nicdom.Nic{}
		nic2.Name = nicName
		nic2.Tenant = tenant
		nic2.Workspace = workspace
		err = readerRepo.Load(ctx, &nic2)
		require.NoError(t, err)

		// Update the NIC: add a security group reference (securityGroupRefs is mutable).
		updateDomain := newNicDomain()
		updateDomain.ResourceVersion = nic2.ResourceVersion
		updateDomain.Spec.SecurityGroupRefs = []commondomain.Reference{{Resource: "sg-1"}}

		updateResult, err := writerRepo.Update(ctx, updateDomain)
		require.NoError(t, err)
		require.NotNil(t, updateResult)
		updated := *updateResult
		require.Len(t, updated.Spec.SecurityGroupRefs, 1)
		require.Equal(t, "sg-1", updated.Spec.SecurityGroupRefs[0].Resource)

		// Verify update with Get
		nic3 := &nicdom.Nic{}
		nic3.Name = nicName
		nic3.Tenant = tenant
		nic3.Workspace = workspace
		err = readerRepo.Load(ctx, &nic3)
		require.NoError(t, err)
		require.Len(t, nic3.Spec.SecurityGroupRefs, 1)
		require.Equal(t, "sg-1", nic3.Spec.SecurityGroupRefs[0].Resource)

		// List NICs and verify ours exists
		var nics []*nicdom.Nic
		_, err = readerRepo.List(ctx, kernelresource.ListParams{Scope: kernelresource.Scope{Tenant: tenant, Workspace: workspace}}, &nics)
		require.NoError(t, err)
		require.NotEmpty(t, nics)
		found := false
		for _, it := range nics {
			if it != nil && it.Name == nicName {
				found = true
				break
			}
		}
		require.True(t, found, "expected NIC to be present in list")

		// Delete the NIC
		del := &nicdom.Nic{}
		del.Name = nicName
		del.Tenant = tenant
		del.Workspace = workspace
		err = writerRepo.Delete(ctx, del)
		require.NoError(t, err)
	})

	t.Run("get_nonexistent_nic", func(t *testing.T) {
		nic := &nicdom.Nic{}
		nic.Name = "missing-nic"
		nic.Tenant = tenant
		nic.Workspace = workspace
		err := readerRepo.Load(ctx, &nic)
		require.Error(t, err)
		domainErr := kernel.AsError(err)
		require.NotNil(t, domainErr)
		require.Equal(t, kernel.KindNotFound, domainErr.Kind)
	})

	t.Run("reject_skuref_mutation", func(t *testing.T) {
		const immutableNicName = "test-nic-immutable"
		immutableNamespace := namespace

		createDomain := &nicdom.Nic{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: immutableNicName},
				Scope:          kernelresource.Scope{Tenant: tenant, Workspace: workspace},
				Labels:         map[string]string{k8slabels.InternalTenantLabel: tenant},
			},
			Spec: nicdom.NicSpec{
				Addresses: []string{"10.0.0.6"},
				SubnetRef: commondomain.Reference{Resource: "subnet-1"},
				SkuRef:    commondomain.Reference{Resource: "sku-original"},
			},
		}

		statusClient, err := dynamic.NewForConfig(testCfg)
		require.NoError(t, err)
		statusCtx, statusCancel := context.WithCancel(ctx)
		defer statusCancel()
		go testutil.SimulateStatusController(statusCtx, statusClient, NICGVR, immutableNamespace, immutableNicName, map[string]interface{}{
			"state": "active",
		})

		_, err = writerRepo.Create(ctx, createDomain)
		require.NoError(t, err)

		current := &nicdom.Nic{}
		current.Name = immutableNicName
		current.Tenant = tenant
		current.Workspace = workspace
		err = readerRepo.Load(ctx, &current)
		require.NoError(t, err)

		// Attempt to change the immutable skuRef — the apiserver CEL rule must reject it.
		mutated := &nicdom.Nic{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name:            immutableNicName,
					ResourceVersion: current.ResourceVersion,
				},
				Scope: kernelresource.Scope{Tenant: tenant, Workspace: workspace},
			},
			Spec: nicdom.NicSpec{
				Addresses: []string{"10.0.0.6"},
				SubnetRef: commondomain.Reference{Resource: "subnet-1"},
				SkuRef:    commondomain.Reference{Resource: "sku-changed"},
			},
		}
		_, err = writerRepo.Update(ctx, mutated)
		require.Error(t, err, "changing the immutable skuRef should be rejected")
		require.ErrorContains(t, err, "spec.skuRef is immutable")
	})
}
