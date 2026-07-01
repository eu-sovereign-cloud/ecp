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
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	"github.com/eu-sovereign-cloud/ecp/resource/common/frontend/testutil"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
)

func TestNetworkBackend_CreateAndGetNetwork(t *testing.T) {
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
	tenant := "t-net-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	const networkName = "test-network"

	// Network CRs live in the namespace computed from tenant (no workspace set in test).
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant})

	ctx := context.Background()

	// Create the namespace before creating network resources. The WriterAdapter
	// does not manage namespaces automatically, so it must exist in advance.
	_, err = k8sadapter.CreateNamespace(ctx, clientset, namespace, map[string]string{
		k8slabels.InternalTenantLabel: tenant,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset, namespace)
	})

	writerRepo := k8sadapter.NewWriterAdapter[*netdom.Network](
		dynClient,
		NetworkGVR,
		slog.Default(),
		NetworkToCR,
		NetworkFromCR,
	)

	readerRepo := k8sadapter.NewReaderAdapter[*netdom.Network](
		dynClient,
		NetworkGVR,
		slog.Default(),
		NetworkFromCR,
	)

	t.Run("create_update_list_delete_network", func(t *testing.T) {
		createDomain := &netdom.Network{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: networkName},
				Scope:          kernelresource.Scope{Tenant: tenant},
				Labels:         map[string]string{k8slabels.InternalTenantLabel: tenant},
			},
			Spec: netdom.NetworkSpec{
				CIDR: netdom.CIDR{
					IPv4: "10.0.0.0/16",
				},
				SkuRef: commondomain.Reference{
					Resource: "standard-network",
				},
			},
		}

		// Simulate a controller that sets status.state after the CR is created.
		// SimulateStatusController is used so that any subsequent Get or Update
		// that relies on status being present will find a valid state.
		statusCfg := rest.CopyConfig(cfg)
		statusCfg.QPS = 50
		statusCfg.Burst = 100
		statusClient, err := dynamic.NewForConfig(statusCfg)
		require.NoError(t, err)

		statusCtx, statusCancel := context.WithCancel(ctx)
		defer statusCancel()
		go testutil.SimulateStatusController(statusCtx, statusClient, NetworkGVR, namespace, networkName, map[string]interface{}{
			"state": "active",
		})

		// Create the network
		result, err := writerRepo.Create(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, result)
		createdDomain := *result
		require.Equal(t, networkName, createdDomain.Name)
		require.Equal(t, "10.0.0.0/16", createdDomain.Spec.CIDR.IPv4)
		require.Equal(t, "standard-network", createdDomain.Spec.SkuRef.Resource)

		// Get the network and verify it matches
		net := &netdom.Network{}
		net.Name = networkName
		net.Tenant = tenant
		err = readerRepo.Load(ctx, &net)
		require.NoError(t, err)
		retrievedDomain := net
		require.NotNil(t, retrievedDomain)
		require.Equal(t, networkName, retrievedDomain.Name)
		require.Equal(t, "10.0.0.0/16", retrievedDomain.Spec.CIDR.IPv4)
		require.Equal(t, "standard-network", retrievedDomain.Spec.SkuRef.Resource)

		// Get current resource version for the update
		net2 := &netdom.Network{}
		net2.Name = networkName
		net2.Tenant = tenant
		err = readerRepo.Load(ctx, &net2)
		require.NoError(t, err)

		// Update the network: add an AdditionalCidr (primary Cidr is immutable in the spec)
		updateDomain := &netdom.Network{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name:            networkName,
					ResourceVersion: net2.ResourceVersion,
				},
				Scope: kernelresource.Scope{Tenant: tenant},
			},
			Spec: netdom.NetworkSpec{
				CIDR: netdom.CIDR{
					IPv4: "10.0.0.0/16",
				},
				SkuRef: commondomain.Reference{
					Resource: "standard-network",
				},
				AdditionalCIDRs: []netdom.CIDR{
					{IPv4: "10.1.0.0/24"},
				},
			},
		}
		updateResult, err := writerRepo.Update(ctx, updateDomain)
		require.NoError(t, err)
		require.NotNil(t, updateResult)
		updated := *updateResult
		require.Len(t, updated.Spec.AdditionalCIDRs, 1)
		require.Equal(t, "10.1.0.0/24", updated.Spec.AdditionalCIDRs[0].IPv4)

		// Verify update with Get
		net3 := &netdom.Network{}
		net3.Name = networkName
		net3.Tenant = tenant
		err = readerRepo.Load(ctx, &net3)
		require.NoError(t, err)
		require.Len(t, net3.Spec.AdditionalCIDRs, 1)
		require.Equal(t, "10.1.0.0/24", net3.Spec.AdditionalCIDRs[0].IPv4)

		// List networks and verify ours exists
		var networks []*netdom.Network
		_, err = readerRepo.List(ctx, kernelresource.ListParams{Scope: kernelresource.Scope{Tenant: tenant}}, &networks)
		require.NoError(t, err)
		require.NotEmpty(t, networks)
		found := false
		for _, it := range networks {
			if it != nil && it.Name == networkName {
				found = true
				break
			}
		}
		require.True(t, found, "expected network to be present in list")

		// Delete the network
		del := &netdom.Network{}
		del.Name = networkName
		del.Tenant = tenant
		err = writerRepo.Delete(ctx, del)
		require.NoError(t, err)
	})

	t.Run("get_nonexistent_network", func(t *testing.T) {
		net := &netdom.Network{}
		net.Name = "missing-network"
		net.Tenant = tenant
		err := readerRepo.Load(ctx, &net)
		require.Error(t, err)
	})
}
