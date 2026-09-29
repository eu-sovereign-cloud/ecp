//go:build envtest

package kubernetes_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	k8sinterface "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
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
	const (
		workspaceName   = "test-workspace"
		workspaceRegion = "region-one"
	)

	writerRepo := k8sadapter.NewNamespaceManagingWriterAdapter[*wsdom.Workspace](
		dynClient,
		clientset,
		WorkspaceGVR,
		slog.Default(),
		Converter,
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

	// lookup is the identity every read, update and delete addresses the workspace by: a
	// workspace is keyed by tenant, name and region.
	lookup := func(name string) *wsdom.Workspace {
		ws := &wsdom.Workspace{}
		ws.Name = name
		ws.Tenant = tenant
		ws.Region = workspaceRegion
		return ws
	}

	// Neither the tenant nor the per-region namespace is pre-created: against a real API
	// server, a write into a missing namespace fails, so create_workspace below passing is the
	// proof that the writer provisions the namespace the Workspace CR itself lives in.
	tenantNamespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant})
	namespace := k8sadapter.ComputeRegionNamespace(lookup(workspaceName))
	childNamespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant, Workspace: workspaceName})

	t.Cleanup(func() {
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset, tenantNamespace)
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset, namespace)
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset, childNamespace)
	})

	t.Run("create_workspace", func(t *testing.T) {
		createDomain := &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: workspaceName},
				Scope:          kernelresource.Scope{Tenant: tenant},
				Region:         workspaceRegion,
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
		ws := lookup(workspaceName)
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
		ws := lookup("missing-workspace")
		err := readerRepo.Load(ctx, &ws)
		require.Error(t, err)
	})

	t.Run("list_workspace", func(t *testing.T) {
		var workspaces []*wsdom.Workspace
		params := regionListParams{ListParams: kernelresource.ListParams{Scope: kernelresource.Scope{Tenant: tenant}}, region: workspaceRegion}
		_, err := readerRepo.List(ctx, params, &workspaces)
		require.NoError(t, err)
		require.Len(t, workspaces, 1)
		require.Equal(t, workspaceName, workspaces[0].Name)
	})

	t.Run("update_workspace", func(t *testing.T) {
		// First get the current resource version
		ws := lookup(workspaceName)
		err := readerRepo.Load(ctx, &ws)
		require.NoError(t, err)

		updateDomain := &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name:            workspaceName,
					ResourceVersion: ws.ResourceVersion,
				},
				Scope:  kernelresource.Scope{Tenant: tenant},
				Region: workspaceRegion,
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
		del := lookup(workspaceName)
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

		del := lookup(workspaceName)

		cleanup := k8sadapter.NamespaceCleanup[*wsdom.Workspace](
			dynClient, clientset, slog.Default(), WorkspaceGVR, k8sadapter.WorkspaceChildren, nil,
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

// regionListParams is the list filter the workspace handler builds: ListParams plus the region
// the request is addressed to, which is what resolves the per-region namespace to list.
type regionListParams struct {
	kernelresource.ListParams
	region string
}

func (p regionListParams) GetRegion() string { return p.region }

// TestWorkspaceRegionIdentity is the whole point of issue #396, against a real API server:
// one tenant can hold a workspace of the same name in two regions, each addressable only
// through its own region, and `region` is a field the CRD schema actually accepts.
func TestWorkspaceRegionIdentity(t *testing.T) {
	t.Parallel()

	testCfg := rest.CopyConfig(cfg)
	testCfg.QPS = 50
	testCfg.Burst = 100

	dynClient, err := dynamic.NewForConfig(testCfg)
	require.NoError(t, err)
	clientset, err := k8sinterface.NewForConfig(testCfg)
	require.NoError(t, err)

	const (
		workspaceName = "shared-name"
		regionOne     = "region-one"
		regionTwo     = "region-two"
	)
	tenant := "t-region-identity"

	writerRepo := k8sadapter.NewNamespaceManagingWriterAdapter[*wsdom.Workspace](
		dynClient, clientset, WorkspaceGVR, slog.Default(), Converter, k8sadapter.WorkspaceChildren, nil,
	)
	readerRepo := k8sadapter.NewReaderAdapter[*wsdom.Workspace](
		dynClient, WorkspaceGVR, slog.Default(), WorkspaceFromCR,
	)

	newWorkspace := func(region string, spec wsdom.WorkspaceSpec) *wsdom.Workspace {
		return &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: workspaceName},
				Scope:          kernelresource.Scope{Tenant: tenant},
				Region:         region,
			},
			Spec: spec,
		}
	}

	ctx := context.Background()
	t.Cleanup(func() {
		for _, region := range []string{regionOne, regionTwo} {
			_ = writerRepo.Delete(context.Background(), newWorkspace(region, nil))
			_ = k8sadapter.DeleteNamespace(context.Background(), clientset,
				k8sadapter.ComputeRegionNamespace(newWorkspace(region, nil)))
		}
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset,
			k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant}))
		_ = k8sadapter.DeleteNamespace(context.Background(), clientset,
			k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant, Workspace: workspaceName}))
	})

	// Neither namespace is pre-created: the write path has to provision the per-region one
	// before the CR, exactly as it does the tenant namespace for a region-less resource.
	t.Run("the same name in two regions is two workspaces", func(t *testing.T) {
		one, err := writerRepo.Create(ctx, newWorkspace(regionOne, wsdom.WorkspaceSpec{"where": regionOne}))
		require.NoError(t, err)
		require.Equal(t, regionOne, (*one).Region)

		two, err := writerRepo.Create(ctx, newWorkspace(regionTwo, wsdom.WorkspaceSpec{"where": regionTwo}))
		require.NoError(t, err, "the second region must not collide with the first")
		require.Equal(t, regionTwo, (*two).Region)
	})

	t.Run("each region reads back its own", func(t *testing.T) {
		for _, region := range []string{regionOne, regionTwo} {
			ws := newWorkspace(region, nil)
			require.NoError(t, readerRepo.Load(ctx, &ws))
			require.Equal(t, region, ws.Region)
			require.Equal(t, region, ws.Spec["where"], "a read in one region must not resolve the other's CR")
		}
	})

	// The region label is still written, so the gateway's server-side list filter works.
	t.Run("the region reaches the CR as a field and a label", func(t *testing.T) {
		cr, err := dynClient.Resource(WorkspaceGVR).
			Namespace(k8sadapter.ComputeRegionNamespace(newWorkspace(regionTwo, nil))).
			Get(ctx, workspaceName, metav1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, regionTwo, cr.Object["region"], "region must survive the CRD schema as a first-class field")
		require.Equal(t, regionTwo, cr.GetLabels()[k8slabels.InternalRegionLabel])
	})

	t.Run("deleting one region leaves the other standing", func(t *testing.T) {
		require.NoError(t, writerRepo.Delete(ctx, newWorkspace(regionOne, nil)))

		gone := newWorkspace(regionOne, nil)
		require.Error(t, readerRepo.Load(ctx, &gone))

		survivor := newWorkspace(regionTwo, nil)
		require.NoError(t, readerRepo.Load(ctx, &survivor))
	})

	// The namespace a workspace owns for its children is hashed from tenant and name alone, so
	// both regions' copies own the same one. Reclaiming it on the first delete would leave the
	// survivor's children resolving a namespace that is gone. Against a real API server this is
	// also what proves the co-owner lookup's metadata.name field selector actually selects.
	t.Run("the shared children namespace goes with the last region deleted", func(t *testing.T) {
		childNamespace := k8sadapter.ComputeNamespace(
			&kernelresource.Scope{Tenant: tenant, Workspace: workspaceName})
		cleanup := k8sadapter.NamespaceCleanup[*wsdom.Workspace](
			dynClient, clientset, slog.Default(), WorkspaceGVR, k8sadapter.WorkspaceChildren, nil,
		)

		// region-one's CR is gone by now, but region-two still owns the namespace.
		require.NoError(t, cleanup(ctx, newWorkspace(regionOne, nil)))
		ns, err := clientset.CoreV1().Namespaces().Get(ctx, childNamespace, metav1.GetOptions{})
		require.NoError(t, err)
		require.Nil(t, ns.DeletionTimestamp, "the surviving region's workspace still needs this namespace")

		require.NoError(t, writerRepo.Delete(ctx, newWorkspace(regionTwo, nil)))
		require.NoError(t, cleanup(ctx, newWorkspace(regionTwo, nil)))

		// envtest runs no namespace controller, so a deleted namespace stays Terminating rather
		// than disappearing — the deletionTimestamp is the proof the delete was accepted.
		after, err := clientset.CoreV1().Namespaces().Get(ctx, childNamespace, metav1.GetOptions{})
		require.NoError(t, err)
		require.NotNil(t, after.DeletionTimestamp, "the last owner deleted must reclaim the namespace")
	})
}

// TestWorkspaceRegionFieldAndLabelInSync holds a Workspace CR's region field and its internal
// region label together against a real API server, with the generated CRD loaded. The field is
// what the plugin provisions into and the namespace is derived from; the label is what the
// gateway's list filter and the delegator's region scope select on. A CR on which they disagree is
// listed and reconciled as one region and created in another, so: the CRD refuses a CR with no
// region and a region change, every adapter write path leaves the two equal, a write that could
// only split them is refused rather than applied, and one that can repair a diverged label does.
func TestWorkspaceRegionFieldAndLabelInSync(t *testing.T) {
	t.Parallel()

	testCfg := rest.CopyConfig(cfg)
	testCfg.QPS = 50
	testCfg.Burst = 100

	dynClient, err := dynamic.NewForConfig(testCfg)
	require.NoError(t, err)
	clientset, err := k8sinterface.NewForConfig(testCfg)
	require.NoError(t, err)

	const (
		workspaceName = "in-sync"
		region        = "region-one"
		otherRegion   = "region-two"
	)
	tenant := "t-region-sync"

	writerRepo := k8sadapter.NewNamespaceManagingWriterAdapter[*wsdom.Workspace](
		dynClient, clientset, WorkspaceGVR, slog.Default(), Converter, k8sadapter.WorkspaceChildren, nil,
	)
	readerRepo := k8sadapter.NewReaderAdapter[*wsdom.Workspace](
		dynClient, WorkspaceGVR, slog.Default(), WorkspaceFromCR,
	)

	newWorkspace := func(name, region string, spec wsdom.WorkspaceSpec) *wsdom.Workspace {
		return &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: name},
				Scope:          kernelresource.Scope{Tenant: tenant},
				Region:         region,
			},
			Spec: spec,
		}
	}
	namespace := k8sadapter.ComputeRegionNamespace(newWorkspace(workspaceName, region, nil))
	crs := dynClient.Resource(WorkspaceGVR).Namespace(namespace)

	ctx := context.Background()
	t.Cleanup(func() {
		for _, name := range []string{workspaceName, "hand-written"} {
			_ = crs.Delete(context.Background(), name, metav1.DeleteOptions{})
		}
		for _, ns := range []string{
			namespace,
			k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant}),
			k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant, Workspace: workspaceName}),
		} {
			_ = k8sadapter.DeleteNamespace(context.Background(), clientset, ns)
		}
	})

	// requireInSync reads the stored CR and fails unless its field and label both say want.
	requireInSync := func(t *testing.T, name, want string) {
		t.Helper()
		cr, err := crs.Get(ctx, name, metav1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, want, cr.Object["region"], "region field")
		require.Equal(t, want, cr.GetLabels()[k8slabels.InternalRegionLabel], "region label")
	}

	t.Run("a workspace without a region is refused", func(t *testing.T) {
		_, err := writerRepo.Create(ctx, newWorkspace(workspaceName, "", nil))
		require.Error(t, err)
		require.Equal(t, kernel.KindValidation, kernel.AsError(err).Kind)
		require.ErrorContains(t, err, "region is required", "the converter refuses it before the CR write")
		require.False(t, apierrors.IsInvalid(err), "the refusal must not be the API server's")

		// Past the converter, the CRD itself refuses a CR with the field missing or empty.
		require.NoError(t, k8sadapter.CreateNamespace(ctx, clientset, namespace, nil))
		raw := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": GroupVersion.String(),
			"kind":       WorkspaceKind,
			"metadata":   map[string]any{"name": "no-region", "namespace": namespace},
		}}
		_, err = crs.Create(ctx, raw, metav1.CreateOptions{})
		require.True(t, apierrors.IsInvalid(err), "a CR with no region field must be invalid, got %v", err)

		raw.Object["region"] = ""
		_, err = crs.Create(ctx, raw, metav1.CreateOptions{})
		require.True(t, apierrors.IsInvalid(err), "a CR with an empty region must be invalid, got %v", err)
	})

	t.Run("every write path leaves field and label equal", func(t *testing.T) {
		_, err := writerRepo.Create(ctx, newWorkspace(workspaceName, region, wsdom.WorkspaceSpec{"step": "create"}))
		require.NoError(t, err)
		requireInSync(t, workspaceName, region)

		// No resourceVersion: the read-modify-write arm, the one that copies fields one by one.
		_, err = writerRepo.Update(ctx, newWorkspace(workspaceName, region, wsdom.WorkspaceSpec{"step": "unversioned"}))
		require.NoError(t, err)
		requireInSync(t, workspaceName, region)

		// With the resourceVersion just read: the full-replace arm, as a plugin writes.
		current := newWorkspace(workspaceName, region, nil)
		require.NoError(t, readerRepo.Load(ctx, &current))
		current.Spec = wsdom.WorkspaceSpec{"step": "versioned"}
		_, err = writerRepo.Update(ctx, current)
		require.NoError(t, err)
		requireInSync(t, workspaceName, region)

		// The status subresource, as the delegator writes it.
		require.NoError(t, readerRepo.Load(ctx, &current))
		current.Status = &wsdom.WorkspaceStatus{}
		current.Status.PushCondition(commondomain.DefaultPendingCondition)
		_, err = writerRepo.UpdateStatus(ctx, current)
		require.NoError(t, err)
		requireInSync(t, workspaceName, region)
	})

	t.Run("the region is immutable", func(t *testing.T) {
		patch := []byte(`{"region":"` + otherRegion + `"}`)
		_, err := crs.Patch(ctx, workspaceName, types.MergePatchType, patch, metav1.PatchOptions{})
		require.True(t, apierrors.IsInvalid(err), "changing the region must be refused, got %v", err)
		require.ErrorContains(t, err, "region is immutable")

		_, err = crs.Patch(ctx, workspaceName, types.MergePatchType, []byte(`{"region":null}`), metav1.PatchOptions{})
		require.True(t, apierrors.IsInvalid(err), "removing the region must be refused, got %v", err)

		requireInSync(t, workspaceName, region)
	})

	// The label is not covered by the CRD — no CRD rule can read metadata.labels — so an edit
	// out of band can still move it. The next write through the adapter puts it back.
	t.Run("the next write repairs a label moved out of band", func(t *testing.T) {
		patch := []byte(`{"metadata":{"labels":{"` + k8slabels.InternalRegionLabel + `":"` + otherRegion + `"}}}`)
		_, err := crs.Patch(ctx, workspaceName, types.MergePatchType, patch, metav1.PatchOptions{})
		require.NoError(t, err)

		ws := newWorkspace(workspaceName, region, nil)
		require.NoError(t, readerRepo.Load(ctx, &ws))
		require.Equal(t, region, ws.Region, "the reader takes the field, never the label")

		_, err = writerRepo.Update(ctx, newWorkspace(workspaceName, region, wsdom.WorkspaceSpec{"step": "repair"}))
		require.NoError(t, err)
		requireInSync(t, workspaceName, region)
	})

	// A CR written by hand into this region's namespace, but naming another region, agrees with
	// itself and not with where it lives. An update addressed to this region would move the
	// label to it; carrying the field along runs into the immutability rule, so the write is
	// refused and the CR is left as it was — never half-moved.
	t.Run("a write that could only split them is refused", func(t *testing.T) {
		stray := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": GroupVersion.String(),
			"kind":       WorkspaceKind,
			"metadata": map[string]any{
				"name":      "hand-written",
				"namespace": namespace,
				"labels": map[string]any{
					k8slabels.InternalTenantLabel: tenant,
					k8slabels.InternalRegionLabel: otherRegion,
				},
			},
			"region": otherRegion,
		}}
		_, err := crs.Create(ctx, stray, metav1.CreateOptions{})
		require.NoError(t, err)

		_, err = writerRepo.Update(ctx, newWorkspace("hand-written", region, wsdom.WorkspaceSpec{"step": "split"}))
		require.Error(t, err)
		require.Equal(t, kernel.KindValidation, kernel.AsError(err).Kind)
		requireInSync(t, "hand-written", otherRegion)
	})
}
