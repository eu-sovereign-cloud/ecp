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

	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/labels"

	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	testutil "github.com/eu-sovereign-cloud/ecp/resources/common/testutil"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/workspace/v1"
	. "github.com/eu-sovereign-cloud/ecp/resources/workspace/v1/backend/kubernetes"
)

func TestWorkspaceBackend(t *testing.T) {
	t.Parallel()

	// Use a config copy with higher rate limits to avoid rate limiter exhaustion
	// during the adapter's status polling loop (10 req/s exceeds default 5 QPS).
	testCfg := rest.CopyConfig(cfg)
	testCfg.QPS = 50
	testCfg.Burst = 100

	dynClient, err := dynamic.NewForConfig(testCfg)
	require.NoError(t, err)

	clientset, err := k8sinterface.NewForConfig(testCfg)
	require.NoError(t, err)

	// Create valid Kubernetes namespace name (lowercase, alphanumeric and hyphens only).
	// Keep tenant short to fit within 63 char label limit.
	tenant := "t-workspace-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	const workspaceName = "test-workspace"

	writerRepo := k8sadapter.NewNamespaceManagingWriterAdapter[*wsdom.Workspace](
		dynClient,
		clientset,
		WorkspaceGVR,
		slog.Default(),
		MapWorkspaceDomainToCR,
		MapCRToWorkspaceDomain,
	)

	readerRepo := k8sadapter.NewReaderAdapter[*wsdom.Workspace](
		dynClient,
		WorkspaceGVR,
		slog.Default(),
		MapCRToWorkspaceDomain,
	)

	ctx := context.Background()

	// Create the tenant namespace before creating workspace resources.
	// The workspace CR is stored in the tenant namespace (hash of tenant),
	// while the NamespaceManagingWriterAdapter only creates the workspace's
	// child resource namespace (hash of tenant/workspace).
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant})
	_, err = k8sadapter.CreateNamespace(ctx, clientset, namespace, map[string]string{
		k8slabels.InternalTenantLabel: tenant,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset, namespace)
	})

	t.Run("create_workspace", func(t *testing.T) {
		createDomain := &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: workspaceName},
				Scope:          kernelresource.Scope{Tenant: tenant},
				Labels:         map[string]string{k8slabels.InternalTenantLabel: tenant},
			},
			Spec: wsdom.WorkspaceSpec{
				"test-string": "test-value",
				"test-number": int64(42),
				"test-bool":   true,
				"test-list":   []string{"a", "b", "c"},
				"test-map": map[string]interface{}{
					"inner-string": "inner-value",
					"inner-number": int64(7),
					"inner-bool":   false,
					"inner-list":   []int64{1, 2, 3},
				},
			},
		}

		// Simulate a controller that sets status.state after the CR is created.
		// NamespaceManagingWriterAdapter.Create polls for status.state to be non-empty;
		// without this, the poll times out because envtest has no real controller.
		statusCfg := rest.CopyConfig(cfg)
		statusCfg.QPS = 50
		statusCfg.Burst = 100
		statusClient, err := dynamic.NewForConfig(statusCfg)
		require.NoError(t, err)

		statusCtx, statusCancel := context.WithCancel(ctx)
		defer statusCancel()
		go testutil.SimulateStatusController(statusCtx, statusClient, WorkspaceGVR, namespace, workspaceName, nil)

		result, err := writerRepo.Create(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, result)
		created := *result
		require.Equal(t, workspaceName, created.Name)
		require.Equal(t, "test-value", created.Spec["test-string"])
		require.Equal(t, int64(42), created.Spec["test-number"])
		require.Equal(t, true, created.Spec["test-bool"])
		require.Equal(t, []interface{}{"a", "b", "c"}, created.Spec["test-list"])
		require.Equal(t, map[string]interface{}{
			"inner-string": "inner-value",
			"inner-number": int64(7),
			"inner-bool":   false,
			"inner-list":   []interface{}{int64(1), int64(2), int64(3)},
		}, created.Spec["test-map"])
	})

	t.Run("get_workspace", func(t *testing.T) {
		ws := &wsdom.Workspace{}
		ws.Name = workspaceName
		ws.Tenant = tenant
		err := readerRepo.Load(ctx, &ws)
		require.NoError(t, err)
		retrieved := ws
		require.NotNil(t, retrieved)
		require.Equal(t, workspaceName, retrieved.Name)
		require.Equal(t, "test-value", retrieved.Spec["test-string"])
		require.Equal(t, int64(42), retrieved.Spec["test-number"])
		require.Equal(t, true, retrieved.Spec["test-bool"])
		require.Equal(t, []interface{}{"a", "b", "c"}, retrieved.Spec["test-list"])
		require.Equal(t, map[string]interface{}{
			"inner-string": "inner-value",
			"inner-number": int64(7),
			"inner-bool":   false,
			"inner-list":   []interface{}{int64(1), int64(2), int64(3)},
		}, retrieved.Spec["test-map"])
	})

	t.Run("get_nonexistent_workspace", func(t *testing.T) {
		ws := &wsdom.Workspace{}
		ws.Name = "missing-workspace"
		ws.Tenant = tenant
		err := readerRepo.Load(ctx, &ws)
		require.Error(t, err)
	})

	t.Run("list_workspace", func(t *testing.T) {
		var workspaces []*wsdom.Workspace
		_, err := readerRepo.List(ctx, kernelresource.ListParams{Scope: kernelresource.Scope{Tenant: tenant}}, &workspaces)
		require.NoError(t, err)
		require.Len(t, workspaces, 1)
		require.Equal(t, workspaceName, workspaces[0].Name)
	})

	t.Run("update_workspace", func(t *testing.T) {
		// First get the current resource version
		ws := &wsdom.Workspace{}
		ws.Name = workspaceName
		ws.Tenant = tenant
		err := readerRepo.Load(ctx, &ws)
		require.NoError(t, err)

		updateDomain := &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name:            workspaceName,
					ResourceVersion: ws.ResourceVersion,
				},
				Scope: kernelresource.Scope{Tenant: tenant},
			},
			Spec: wsdom.WorkspaceSpec{
				"test-string": "updated-value",
				"test-number": int64(84),
			},
		}

		result, err := writerRepo.Update(ctx, updateDomain)
		require.NoError(t, err)
		require.NotNil(t, result)
		updated := *result
		require.Equal(t, "updated-value", updated.Spec["test-string"])
		require.Equal(t, int64(84), updated.Spec["test-number"])
		require.Nil(t, updated.Spec["test-bool"])
		require.Nil(t, updated.Spec["test-list"])
		require.Nil(t, updated.Spec["test-map"])
	})

	t.Run("delete_workspace", func(t *testing.T) {
		del := &wsdom.Workspace{}
		del.Name = workspaceName
		del.Tenant = tenant
		err := writerRepo.Delete(ctx, del)
		require.NoError(t, err)
	})
}
