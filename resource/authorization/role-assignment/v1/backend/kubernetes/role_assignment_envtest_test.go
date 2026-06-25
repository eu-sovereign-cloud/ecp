//go:build envtest

package kubernetes_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role-assignment/v1"
	. "github.com/eu-sovereign-cloud/ecp/resource/authorization/role-assignment/v1/backend/kubernetes"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	testutil "github.com/eu-sovereign-cloud/ecp/resource/common/frontend/testutil"
)

func TestRoleAssignmentBackend_CreateAndGetRoleAssignment(t *testing.T) {
	t.Parallel()

	// Use a config copy with higher rate limits to avoid rate limiter exhaustion
	// during the adapter's status polling loop.
	testCfg := rest.CopyConfig(cfg)
	testCfg.QPS = 50
	testCfg.Burst = 100
	dynClient, err := dynamic.NewForConfig(testCfg)
	require.NoError(t, err)

	// Create valid Kubernetes namespace name (lowercase, alphanumeric and hyphens only).
	// Keep tenant short to fit within 63 char label limit.
	tenant := "t-ra-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant})
	const roleAssignmentName = "test-role-assignment"
	namespaceGVR := k8sschema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

	// Create the namespace so the CRD resources have somewhere to land.
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

	// Cleanup namespace and all resources within it.
	t.Cleanup(func() {
		_ = dynClient.Resource(namespaceGVR).Delete(context.Background(), namespace, metav1.DeleteOptions{})
	})

	// Build writer and reader adapters directly from the kubernetes backend package.
	writerRepo := k8sadapter.NewWriterAdapter[*radom.RoleAssignment](
		dynClient,
		RoleAssignmentGVR,
		slog.Default(),
		RoleAssignmentToCR,
		RoleAssignmentFromCR,
	)
	readerRepo := k8sadapter.NewReaderAdapter[*radom.RoleAssignment](
		dynClient,
		RoleAssignmentGVR,
		slog.Default(),
		RoleAssignmentFromCR,
	)

	t.Run("create_update_list_delete_role_assignment", func(t *testing.T) {
		createDomain := &radom.RoleAssignment{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: roleAssignmentName},
				Scope:          kernelresource.Scope{Tenant: tenant},
				Labels:         map[string]string{k8slabels.InternalTenantLabel: tenant},
			},
			Spec: radom.RoleAssignmentSpec{
				Subs: []string{"user1@example.com"},
				Scopes: []radom.RoleAssignmentScope{
					{Tenants: []string{tenant}},
				},
				Roles: []string{"workspace-viewer"},
			},
		}

		// Simulate a status controller so the CR's status.state becomes non-empty.
		// WriterAdapter.Create polls for status.state; without this the poll times out
		// because envtest has no real controller.
		statusCfg := rest.CopyConfig(cfg)
		statusCfg.QPS = 50
		statusCfg.Burst = 100
		statusClient, err := dynamic.NewForConfig(statusCfg)
		require.NoError(t, err)

		statusCtx, statusCancel := context.WithCancel(ctx)
		defer statusCancel()
		go testutil.SimulateStatusController(statusCtx, statusClient, RoleAssignmentGVR, namespace, roleAssignmentName, map[string]interface{}{})

		// Create the role assignment.
		resultPtr, err := writerRepo.Create(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, resultPtr)
		created := *resultPtr
		require.Equal(t, roleAssignmentName, created.Name)
		require.Equal(t, []string{"workspace-viewer"}, created.Spec.Roles)
		require.Equal(t, []string{"user1@example.com"}, created.Spec.Subs)

		// Get the role assignment and verify it matches.
		getDomain := &radom.RoleAssignment{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: roleAssignmentName},
				Scope:          kernelresource.Scope{Tenant: tenant},
			},
		}
		err = readerRepo.Load(ctx, &getDomain)
		require.NoError(t, err)
		require.NotNil(t, getDomain)
		require.Equal(t, roleAssignmentName, getDomain.Name)
		require.Equal(t, []string{"workspace-viewer"}, getDomain.Spec.Roles)

		// Update the role assignment spec.
		createDomain.Spec.Roles = []string{"workspace-editor"}
		createDomain.ResourceVersion = created.ResourceVersion
		updatedPtr, err := writerRepo.Update(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, updatedPtr)
		updated := *updatedPtr
		require.Equal(t, []string{"workspace-editor"}, updated.Spec.Roles)

		// Verify update with a Get.
		getDomain2 := &radom.RoleAssignment{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: roleAssignmentName},
				Scope:          kernelresource.Scope{Tenant: tenant},
			},
		}
		err = readerRepo.Load(ctx, &getDomain2)
		require.NoError(t, err)
		require.Equal(t, []string{"workspace-editor"}, getDomain2.Spec.Roles)

		// List role assignments and verify ours exists.
		var items []*radom.RoleAssignment
		listParams := kernelresource.ListParams{Scope: kernelresource.Scope{Tenant: tenant}}
		_, err = readerRepo.List(ctx, listParams, &items)
		require.NoError(t, err)
		require.NotEmpty(t, items)
		found := false
		for _, it := range items {
			if it != nil && it.Name == roleAssignmentName {
				found = true
				break
			}
		}
		require.True(t, found, "expected role assignment to be present in list")

		// Delete the role assignment.
		delDomain := &radom.RoleAssignment{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: roleAssignmentName},
				Scope:          kernelresource.Scope{Tenant: tenant},
			},
		}
		err = writerRepo.Delete(ctx, delDomain)
		require.NoError(t, err)
	})

	t.Run("get_nonexistent_role_assignment", func(t *testing.T) {
		getDomain := &radom.RoleAssignment{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: "missing-role-assignment"},
				Scope:          kernelresource.Scope{Tenant: tenant},
			},
		}
		err := readerRepo.Load(ctx, &getDomain)
		require.Error(t, err)
	})
}
