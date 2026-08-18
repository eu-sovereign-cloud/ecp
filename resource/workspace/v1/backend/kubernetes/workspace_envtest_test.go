//go:build envtest

package kubernetes_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	k8sinterface "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	"github.com/eu-sovereign-cloud/ecp/resource/common/frontend/testutil"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	. "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
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
		WorkspaceToCR,
		WorkspaceFromCR,
		k8sadapter.WorkspaceChildren,
		nil, // no child GVRs in this suite; emptiness check is a no-op
	)

	readerRepo := k8sadapter.NewReaderAdapter[*wsdom.Workspace](
		dynClient,
		WorkspaceGVR,
		slog.Default(),
		WorkspaceFromCR,
	)

	ctx := context.Background()

	// The tenant namespace is deliberately NOT pre-created: against a real API server, a write
	// into a missing namespace fails, so create_workspace below passing is the proof that the
	// writer provisions the namespace the Workspace CR itself lives in.
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant})
	childNamespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant, Workspace: workspaceName})

	t.Cleanup(func() {
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset, namespace)
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset, childNamespace)
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

		// The write path deletes the CR only. Teardown moved to the controller finalizer, so
		// the namespace must still be here.
		_, err = clientset.CoreV1().Namespaces().Get(ctx, childNamespace, metav1.GetOptions{})
		require.NoError(t, err, "the write path must leave the child namespace to the controller")
	})

	// The teardown half, against a real API server: the fake-client unit tests cannot catch a
	// label key the write path and the ownership check disagree on, because both read the same
	// constant. Here the labels make a full round trip through the API server.
	t.Run("cleanup_deletes_the_owned_child_namespace", func(t *testing.T) {
		ns, err := clientset.CoreV1().Namespaces().Get(ctx, childNamespace, metav1.GetOptions{})
		require.NoError(t, err, "create_workspace must have provisioned the child namespace")
		require.Equal(t, tenant, ns.Labels[k8slabels.InternalTenantLabel])
		require.Equal(t, workspaceName, ns.Labels[k8slabels.InternalWorkspaceLabel])

		del := &wsdom.Workspace{}
		del.Name = workspaceName
		del.Tenant = tenant

		cleanup := k8sadapter.NamespaceCleanup[*wsdom.Workspace](
			dynClient, clientset, slog.Default(), k8sadapter.WorkspaceChildren, nil,
		)
		require.NoError(t, cleanup(ctx, del))

		// envtest runs no namespace controller, so a deleted namespace stays Terminating rather
		// than disappearing — the deletionTimestamp is the proof the delete was accepted.
		after, err := clientset.CoreV1().Namespaces().Get(ctx, childNamespace, metav1.GetOptions{})
		require.NoError(t, err)
		require.NotNil(t, after.DeletionTimestamp, "the owned child namespace must be deleted")

		// Idempotent: the finalizer replays this hook whenever dropping it conflicts.
		require.NoError(t, cleanup(ctx, del), "cleanup must tolerate a namespace already deleted")
	})
}
