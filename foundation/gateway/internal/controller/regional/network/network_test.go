package network

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/controller/testutil"
	model "github.com/eu-sovereign-cloud/ecp/foundation/models"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/scope"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/adapters/kubernetes"
	persistencenetwork "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/network"
	networksv1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/network/networks/v1"

	kubernetes2domain "github.com/eu-sovereign-cloud/ecp/foundation/persistence/adapters/kubernetes2domain"
)

func TestNetworkController_CreateAndGetNetwork(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, persistencenetwork.AddToScheme(scheme))

	// Use a config copy with higher rate limits to avoid rate limiter exhaustion
	// during the adapter's status polling loop.
	testCfg := rest.CopyConfig(cfg)
	testCfg.QPS = 50
	testCfg.Burst = 100
	dynClient, err := dynamic.NewForConfig(testCfg)
	require.NoError(t, err)

	// Create valid Kubernetes namespace name (lowercase, alphanumeric and hyphens only)
	// Keep tenant short to fit within 63 char label limit
	tenant := "t-net-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	namespace := kubernetes.ComputeNamespace(&scope.Scope{Tenant: tenant})
	const networkName = "test-network"
	namespaceGVR := k8sschema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

	// Create the namespace object
	namespaceObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": namespace,
			},
		},
	}

	ctx := context.Background()
	_, err = dynClient.Resource(namespaceGVR).Create(ctx, namespaceObj, metav1.CreateOptions{})
	require.NoError(t, err)

	// Cleanup namespace and all resources within it
	t.Cleanup(func() {
		_ = dynClient.Resource(namespaceGVR).Delete(context.Background(), namespace, metav1.DeleteOptions{})
	})

	// Setup controllers
	createController := CreateNetwork{
		Logger: slog.Default(),
		NetworkRepo: kubernetes.NewWriterAdapter(
			dynClient,
			networksv1.NetworkGVR,
			slog.Default(),
			kubernetes2domain.MapNetworkDomainToCR,
			kubernetes2domain.MapCRToNetworkDomain,
		),
	}

	getController := GetNetwork{
		Logger: slog.Default(),
		NetworkRepo: kubernetes.NewReaderAdapter(
			dynClient,
			networksv1.NetworkGVR,
			slog.Default(),
			kubernetes2domain.MapCRToNetworkDomain,
		),
	}

	updateController := UpdateNetwork{
		Logger: slog.Default(),
		NetworkRepo: kubernetes.NewWriterAdapter(
			dynClient,
			networksv1.NetworkGVR,
			slog.Default(),
			kubernetes2domain.MapNetworkDomainToCR,
			kubernetes2domain.MapCRToNetworkDomain,
		),
	}

	deleteController := DeleteNetwork{
		Logger: slog.Default(),
		NetworkRepo: kubernetes.NewWriterAdapter(
			dynClient,
			networksv1.NetworkGVR,
			slog.Default(),
			kubernetes2domain.MapNetworkDomainToCR,
			kubernetes2domain.MapCRToNetworkDomain,
		),
	}

	listController := ListNetworks{
		Logger: slog.Default(),
		NetworkRepo: kubernetes.NewReaderAdapter(
			dynClient,
			networksv1.NetworkGVR,
			slog.Default(),
			kubernetes2domain.MapCRToNetworkDomain,
		),
	}

	t.Run("create_update_list_delete_network", func(t *testing.T) {
		// Create a network domain object
		createDomain := &regional.NetworkDomain{
			Metadata: regional.Metadata{
				CommonMetadata: model.CommonMetadata{
					Name: networkName,
				},
				Scope: scope.Scope{
					Tenant: tenant,
				},
				Labels: map[string]string{
					TenantLabelKey: tenant,
				},
			},
			Spec: regional.NetworkSpecDomain{
				Cidr: regional.CidrDomain{
					IPv4: "10.0.0.0/16",
				},
				SkuRef: regional.ReferenceDomain{
					Resource: "standard-network",
				},
				RouteTableRef: regional.ReferenceDomain{
					Resource: "default-route-table",
				},
			},
		}

		// Simulate a controller that sets status.state after the CR is created.
		// WriterAdapter.Create does not poll for status; SimulateStatusController
		// is used here so that any subsequent Get or Update that relies on status
		// being present will find a valid state.
		statusCfg := rest.CopyConfig(cfg)
		statusCfg.QPS = 50
		statusCfg.Burst = 100
		statusClient, err := dynamic.NewForConfig(statusCfg)
		require.NoError(t, err)

		statusCtx, statusCancel := context.WithCancel(ctx)
		defer statusCancel()
		go testutil.SimulateStatusController(statusCtx, statusClient, networksv1.NetworkGVR, namespace, networkName, map[string]interface{}{
			"state": "active",
		})

		// Create the network
		createdDomain, err := createController.Do(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, createdDomain)
		require.Equal(t, networkName, createdDomain.Name)
		require.Equal(t, "10.0.0.0/16", createdDomain.Spec.Cidr.IPv4)
		require.Equal(t, "standard-network", createdDomain.Spec.SkuRef.Resource)

		// Get the network and verify it matches
		metadata := regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: networkName,
			},
			Scope: scope.Scope{
				Tenant: tenant,
			},
		}
		retrievedDomain, err := getController.Do(ctx, &metadata)
		require.NoError(t, err)
		require.NotNil(t, retrievedDomain)
		require.Equal(t, networkName, retrievedDomain.Name)
		require.Equal(t, "10.0.0.0/16", retrievedDomain.Spec.Cidr.IPv4)
		require.Equal(t, "standard-network", retrievedDomain.Spec.SkuRef.Resource)

		// Update the network: add an AdditionalCidr (primary Cidr is immutable in the spec)
		createDomain.Spec.AdditionalCidrs = []regional.CidrDomain{
			{IPv4: "10.1.0.0/24"},
		}
		updatedDomain, err := updateController.Do(ctx, createDomain)
		require.NoError(t, err)
		require.Len(t, updatedDomain.Spec.AdditionalCidrs, 1)
		require.Equal(t, "10.1.0.0/24", updatedDomain.Spec.AdditionalCidrs[0].IPv4)

		// Verify update with Get
		retrievedDomain, err = getController.Do(ctx, &metadata)
		require.NoError(t, err)
		require.Len(t, retrievedDomain.Spec.AdditionalCidrs, 1)
		require.Equal(t, "10.1.0.0/24", retrievedDomain.Spec.AdditionalCidrs[0].IPv4)

		// List networks and verify ours exists
		listParams := model.ListParams{Scope: scope.Scope{Tenant: tenant}}
		items, _, err := listController.Do(ctx, listParams)
		require.NoError(t, err)
		require.NotEmpty(t, items)
		found := false
		for _, it := range items {
			if it != nil && it.Name == networkName {
				found = true
				break
			}
		}
		require.True(t, found, "expected network to be present in list")

		// Delete the network
		err = deleteController.Do(ctx, &metadata)
		require.NoError(t, err)
	})

	t.Run("get_nonexistent_network", func(t *testing.T) {
		metadata := regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "missing-network",
			},
			Scope: scope.Scope{
				Tenant: tenant,
			},
		}
		_, err := getController.Do(ctx, &metadata)
		require.Error(t, err)
	})
}
