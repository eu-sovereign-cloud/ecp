package kubernetes

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

type fakeNetworkScope struct {
	tenant, workspace, network string
}

func (s fakeNetworkScope) GetTenant() string    { return s.tenant }
func (s fakeNetworkScope) GetWorkspace() string { return s.workspace }
func (s fakeNetworkScope) GetNetwork() string   { return s.network }

func TestComputeNetworkNamespace_DistinctFromComputeNamespace(t *testing.T) {
	scope := kernelresource.Scope{Tenant: "t1", Workspace: "w1"}
	networkScope := fakeNetworkScope{tenant: "t1", workspace: "w1", network: "n1"}

	require.NotEqual(t, ComputeNamespace(&scope), ComputeNetworkNamespace(networkScope),
		"a network-scoped namespace must not collide with the workspace-only namespace for the same tenant/workspace")
}

func TestComputeNetworkNamespace_DistinctPerNetwork(t *testing.T) {
	base := fakeNetworkScope{tenant: "t1", workspace: "w1", network: "n1"}
	other := fakeNetworkScope{tenant: "t1", workspace: "w1", network: "n2"}

	require.NotEqual(t, ComputeNetworkNamespace(base), ComputeNetworkNamespace(other),
		"different networks in the same tenant/workspace must get distinct namespaces")
}

func TestComputeNetworkNamespace_Deterministic(t *testing.T) {
	// Two separately-built scopes with equal fields: passing the same value twice would assert
	// nothing, since a function that returned a fresh random string each call would still pass.
	scope := fakeNetworkScope{tenant: "t1", workspace: "w1", network: "n1"}
	same := fakeNetworkScope{tenant: "t1", workspace: "w1", network: "n1"}

	require.Equal(t, ComputeNetworkNamespace(scope), ComputeNetworkNamespace(same),
		"equal tenant/workspace/network must always map to the same namespace")
}

var testGVR = schema.GroupVersionResource{Group: "network.test", Version: "v1", Resource: "routetables"}

func testListKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{testGVR: "RouteTableList"}
}

func newTestObject(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "network.test/v1",
		"kind":       "RouteTable",
		"metadata":   map[string]any{"namespace": namespace, "name": name},
	}}
}

type testIdentifiable struct {
	name string
}

func (t *testIdentifiable) GetName() string      { return t.name }
func (t *testIdentifiable) GetVersion() string   { return "" }
func (t *testIdentifiable) GetTenant() string    { return "" }
func (t *testIdentifiable) GetWorkspace() string { return "" }

// testNetworkListParams mirrors how a network-scoped resource (e.g. route-table) carries the
// Network dimension on its own local params type instead of on the shared ListParams struct.
type testNetworkListParams struct {
	kernelresource.ListParams
	Network string
}

func (p testNetworkListParams) GetNetwork() string { return p.Network }

// TestReaderAdapter_List_NetworkScopeFallback is a regression test proving that List resolves
// the plain workspace namespace for ordinary resource.ListParams callers (the ListFilter
// interface has no Network dimension), while resolving the network-scoped namespace when a
// caller's params type opts in by also implementing NetworkScope.
func TestReaderAdapter_List_NetworkScopeFallback(t *testing.T) {
	workspaceNS := ComputeNamespace(&kernelresource.Scope{Tenant: "t1", Workspace: "w1"})
	networkNS := ComputeNetworkNamespace(fakeNetworkScope{tenant: "t1", workspace: "w1", network: "n1"})
	require.NotEqual(t, workspaceNS, networkNS)

	workspaceObj := newTestObject(workspaceNS, "in-workspace-ns")
	networkObj := newTestObject(networkNS, "in-network-ns")

	dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), testListKinds(), workspaceObj, networkObj)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	reader := NewReaderAdapter[*testIdentifiable](dynFake, testGVR, logger, func(obj client.Object) (*testIdentifiable, error) {
		return &testIdentifiable{name: obj.GetName()}, nil
	})

	t.Run("unchanged behavior when Network is empty", func(t *testing.T) {
		var out []*testIdentifiable
		_, err := reader.List(context.Background(), kernelresource.ListParams{
			Scope: kernelresource.Scope{Tenant: "t1", Workspace: "w1"},
		}, &out)
		require.NoError(t, err)
		require.Len(t, out, 1)
		require.Equal(t, "in-workspace-ns", out[0].name)
	})

	t.Run("resolves network namespace when params also implements NetworkScope", func(t *testing.T) {
		var out []*testIdentifiable
		_, err := reader.List(context.Background(), testNetworkListParams{
			ListParams: kernelresource.ListParams{Scope: kernelresource.Scope{Tenant: "t1", Workspace: "w1"}},
			Network:    "n1",
		}, &out)
		require.NoError(t, err)
		require.Len(t, out, 1)
		require.Equal(t, "in-network-ns", out[0].name)
	})
}

// testNetworkIdentifiable is a network-scoped domain stand-in (mirroring
// RegionalNetworkMetadata's GetTenant/GetWorkspace/GetNetwork shape) used to prove that every
// adapter method — not just List — resolves the network-scoped namespace.
type testNetworkIdentifiable struct {
	name, tenant, workspace, network string
}

func (t *testNetworkIdentifiable) GetName() string      { return t.name }
func (t *testNetworkIdentifiable) GetVersion() string   { return "" }
func (t *testNetworkIdentifiable) GetTenant() string    { return t.tenant }
func (t *testNetworkIdentifiable) GetWorkspace() string { return t.workspace }
func (t *testNetworkIdentifiable) GetNetwork() string   { return t.network }

// networkDomainToK8s mirrors RouteTableToCR: it stamps the object's own namespace via
// ComputeNetworkNamespace, exactly like the real conversion does.
func networkDomainToK8s(m *testNetworkIdentifiable) (client.Object, error) {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "network.test/v1",
		"kind":       "RouteTable",
		"metadata": map[string]any{
			"namespace": ComputeNetworkNamespace(m),
			"name":      m.name,
		},
	}}, nil
}

func networkK8sToDomain(obj client.Object) (*testNetworkIdentifiable, error) {
	return &testNetworkIdentifiable{name: obj.GetName()}, nil
}

// TestAdapter_NetworkIsolation_AcrossAllOperations is a regression test proving that
// Create/Load (and by the same ComputeNamespace code path, Update/UpdateStatus/Delete) resolve
// the network-scoped namespace, not just List. Two resources sharing the same tenant/workspace
// but different networks must not collide: before this fix, both would have been addressed via
// the workspace-only hash, so the second Create below would have failed with AlreadyExists
// (same name, same — wrong — namespace).
func TestAdapter_NetworkIsolation_AcrossAllOperations(t *testing.T) {
	dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), testListKinds())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	writer := NewWriterAdapter[*testNetworkIdentifiable](dynFake, testGVR, logger, networkDomainToK8s, networkK8sToDomain)
	reader := NewReaderAdapter[*testNetworkIdentifiable](dynFake, testGVR, logger, networkK8sToDomain)

	inN1 := &testNetworkIdentifiable{name: "rt1", tenant: "t1", workspace: "w1", network: "n1"}
	inN2 := &testNetworkIdentifiable{name: "rt1", tenant: "t1", workspace: "w1", network: "n2"}

	_, err := writer.Create(context.Background(), inN1)
	require.NoError(t, err, "creating rt1 in network n1 must succeed")

	_, err = writer.Create(context.Background(), inN2)
	require.NoError(t, err, "creating the same-named rt1 in a different network (n2) must not collide with n1's namespace")

	loadedN1 := &testNetworkIdentifiable{name: "rt1", tenant: "t1", workspace: "w1", network: "n1"}
	require.NoError(t, reader.Load(context.Background(), &loadedN1))
	require.Equal(t, "rt1", loadedN1.name)

	loadedN2 := &testNetworkIdentifiable{name: "rt1", tenant: "t1", workspace: "w1", network: "n2"}
	require.NoError(t, reader.Load(context.Background(), &loadedN2))
	require.Equal(t, "rt1", loadedN2.name)

	require.NoError(t, writer.Delete(context.Background(), inN1), "deleting rt1 in network n1 must target n1's namespace")

	// n2's rt1 must survive n1's deletion — proving Delete is also network-scoped.
	stillThere := &testNetworkIdentifiable{name: "rt1", tenant: "t1", workspace: "w1", network: "n2"}
	require.NoError(t, reader.Load(context.Background(), &stillThere))
	require.Equal(t, "rt1", stillThere.name)
}

// testWorkspaceScopedIdentifiable mirrors Network's domain shape: tenant+workspace scoped,
// with no Network field of its own — its own name becomes the network segment for its
// children (RouteTable, Subnet), the same relationship Workspace's own name has to the
// workspace segment for its children.
type testWorkspaceScopedIdentifiable struct {
	name, tenant, workspace string
}

func (t *testWorkspaceScopedIdentifiable) GetName() string      { return t.name }
func (t *testWorkspaceScopedIdentifiable) GetVersion() string   { return "" }
func (t *testWorkspaceScopedIdentifiable) GetTenant() string    { return t.tenant }
func (t *testWorkspaceScopedIdentifiable) GetWorkspace() string { return t.workspace }

func TestChildNamespaceFor_WorkspaceChildren(t *testing.T) {
	net := &testWorkspaceScopedIdentifiable{name: "net1", tenant: "t1", workspace: "w1"}

	namespace, ownerLabels := childNamespaceFor(WorkspaceChildren, net)

	require.Equal(t, ComputeNamespace(&kernelresource.Scope{Tenant: "t1", Workspace: "net1"}), namespace,
		"WorkspaceChildren must provision the namespace hashing tenant + the resource's own name as the workspace segment")
	require.Equal(t, map[string]string{
		labels.InternalTenantLabel:    "t1",
		labels.InternalWorkspaceLabel: "net1",
	}, ownerLabels)
}

func TestChildNamespaceFor_NetworkChildren(t *testing.T) {
	net := &testWorkspaceScopedIdentifiable{name: "net1", tenant: "t1", workspace: "w1"}

	namespace, ownerLabels := childNamespaceFor(NetworkChildren, net)

	require.Equal(t, ComputeNetworkNamespace(fakeNetworkScope{tenant: "t1", workspace: "w1", network: "net1"}), namespace,
		"NetworkChildren must provision the namespace hashing tenant/workspace + the resource's own name as the network segment")
	require.NotEqual(t, ComputeNamespace(&kernelresource.Scope{Tenant: "t1", Workspace: "w1"}), namespace,
		"the provisioned namespace must be distinct from the Network's own (workspace-level) namespace")
	require.Equal(t, map[string]string{
		labels.InternalTenantLabel:    "t1",
		labels.InternalWorkspaceLabel: "w1",
		labels.InternalNetworkLabel:   "net1",
	}, ownerLabels)
}

func TestChildNamespaceFor_NoChildNamespace(t *testing.T) {
	net := &testWorkspaceScopedIdentifiable{name: "net1", tenant: "t1", workspace: "w1"}

	namespace, ownerLabels := childNamespaceFor(NoChildNamespace, net)

	require.Empty(t, namespace)
	require.Nil(t, ownerLabels)
}

// --- NamespaceManagingWriterAdapter.Delete: empty-check + owned NS cleanup ---

var (
	testParentGVR = schema.GroupVersionResource{Group: "workspace.test", Version: "v1", Resource: "workspaces"}
	testChildGVR  = schema.GroupVersionResource{Group: "storage.test", Version: "v1", Resource: "blockstorages"}
)

func parentListKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{
		testParentGVR: "WorkspaceList",
		testChildGVR:  "BlockStorageList",
	}
}

// parentDomainToK8s places the parent CR in the tenant namespace (like WorkspaceToCR).
func parentDomainToK8s(m *testWorkspaceScopedIdentifiable) (client.Object, error) {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "workspace.test/v1",
		"kind":       "Workspace",
		"metadata": map[string]any{
			"namespace": ComputeNamespace(&kernelresource.Scope{Tenant: m.tenant}),
			"name":      m.name,
		},
	}}, nil
}

func parentK8sToDomain(obj client.Object) (*testWorkspaceScopedIdentifiable, error) {
	return &testWorkspaceScopedIdentifiable{name: obj.GetName(), tenant: "t1"}, nil
}

func newChildObject(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "storage.test/v1",
		"kind":       "BlockStorage",
		"metadata":   map[string]any{"namespace": namespace, "name": name},
	}}
}

func TestNamespaceHasChildResources(t *testing.T) {
	childNS := ComputeNamespace(&kernelresource.Scope{Tenant: "t1", Workspace: "w1"})
	gvrs := []schema.GroupVersionResource{testChildGVR}

	t.Run("empty namespace", func(t *testing.T) {
		dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), parentListKinds())
		has, err := namespaceHasChildResources(context.Background(), dynFake, childNS, gvrs)
		require.NoError(t, err)
		require.False(t, has)
	})

	t.Run("namespace with a child resource", func(t *testing.T) {
		dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(
			runtime.NewScheme(), parentListKinds(), newChildObject(childNS, "bs-1"),
		)
		has, err := namespaceHasChildResources(context.Background(), dynFake, childNS, gvrs)
		require.NoError(t, err)
		require.True(t, has)
	})

	t.Run("nil gvrs means empty", func(t *testing.T) {
		dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(
			runtime.NewScheme(), parentListKinds(), newChildObject(childNS, "bs-1"),
		)
		has, err := namespaceHasChildResources(context.Background(), dynFake, childNS, nil)
		require.NoError(t, err)
		require.False(t, has)
	})

	t.Run("empty namespace name means empty", func(t *testing.T) {
		dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), parentListKinds())
		has, err := namespaceHasChildResources(context.Background(), dynFake, "", gvrs)
		require.NoError(t, err)
		require.False(t, has)
	})
}

func TestNamespaceManagingWriterAdapter_Delete(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parent := &testWorkspaceScopedIdentifiable{name: "w1", tenant: "t1"}
	// WorkspaceChildren uses GetName() as the workspace segment.
	childNS := ComputeNamespace(&kernelresource.Scope{Tenant: "t1", Workspace: "w1"})
	tenantNS := ComputeNamespace(&kernelresource.Scope{Tenant: "t1"})
	ownerLabels := map[string]string{
		labels.InternalTenantLabel:    "t1",
		labels.InternalWorkspaceLabel: "w1",
	}

	t.Run("refuses delete when child namespace has SECA resources", func(t *testing.T) {
		parentObj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "workspace.test/v1",
			"kind":       "Workspace",
			"metadata":   map[string]any{"namespace": tenantNS, "name": "w1"},
		}}
		dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(
			runtime.NewScheme(), parentListKinds(), parentObj, newChildObject(childNS, "bs-1"),
		)
		cs := k8sfake.NewClientset(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: childNS, Labels: ownerLabels},
		})

		writer := NewNamespaceManagingWriterAdapter[*testWorkspaceScopedIdentifiable](
			dynFake, cs, testParentGVR, logger, parentDomainToK8s, parentK8sToDomain,
			WorkspaceChildren, []schema.GroupVersionResource{testChildGVR},
		)

		err := writer.Delete(context.Background(), parent)
		require.Error(t, err)
		var domainErr *kernel.Error
		require.ErrorAs(t, err, &domainErr)
		require.Equal(t, kernel.KindConflict, domainErr.Kind)

		// Parent CR must still exist.
		_, getErr := dynFake.Resource(testParentGVR).Namespace(tenantNS).Get(
			context.Background(), "w1", metav1.GetOptions{},
		)
		require.NoError(t, getErr)

		// Child namespace must still exist.
		_, nsErr := cs.CoreV1().Namespaces().Get(context.Background(), childNS, metav1.GetOptions{})
		require.NoError(t, nsErr)
	})

	t.Run("deletes parent and owned empty child namespace", func(t *testing.T) {
		parentObj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "workspace.test/v1",
			"kind":       "Workspace",
			"metadata":   map[string]any{"namespace": tenantNS, "name": "w1"},
		}}
		dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(
			runtime.NewScheme(), parentListKinds(), parentObj,
		)
		cs := k8sfake.NewClientset(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: childNS, Labels: ownerLabels},
		})

		writer := NewNamespaceManagingWriterAdapter[*testWorkspaceScopedIdentifiable](
			dynFake, cs, testParentGVR, logger, parentDomainToK8s, parentK8sToDomain,
			WorkspaceChildren, []schema.GroupVersionResource{testChildGVR},
		)

		require.NoError(t, writer.Delete(context.Background(), parent))

		_, getErr := dynFake.Resource(testParentGVR).Namespace(tenantNS).Get(
			context.Background(), "w1", metav1.GetOptions{},
		)
		require.True(t, kerrs.IsNotFound(getErr), "parent CR should be deleted")

		_, nsErr := cs.CoreV1().Namespaces().Get(context.Background(), childNS, metav1.GetOptions{})
		require.True(t, kerrs.IsNotFound(nsErr), "owned empty child namespace should be deleted")
	})

	t.Run("does not delete child namespace that is not owned", func(t *testing.T) {
		parentObj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "workspace.test/v1",
			"kind":       "Workspace",
			"metadata":   map[string]any{"namespace": tenantNS, "name": "w1"},
		}}
		dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(
			runtime.NewScheme(), parentListKinds(), parentObj,
		)
		// Namespace exists but lacks ownership labels.
		cs := k8sfake.NewClientset(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: childNS},
		})

		writer := NewNamespaceManagingWriterAdapter[*testWorkspaceScopedIdentifiable](
			dynFake, cs, testParentGVR, logger, parentDomainToK8s, parentK8sToDomain,
			WorkspaceChildren, []schema.GroupVersionResource{testChildGVR},
		)

		require.NoError(t, writer.Delete(context.Background(), parent))

		_, getErr := dynFake.Resource(testParentGVR).Namespace(tenantNS).Get(
			context.Background(), "w1", metav1.GetOptions{},
		)
		require.True(t, kerrs.IsNotFound(getErr))

		_, nsErr := cs.CoreV1().Namespaces().Get(context.Background(), childNS, metav1.GetOptions{})
		require.NoError(t, nsErr, "unowned namespace must not be deleted")
	})

	t.Run("NoChildNamespace skips empty check and namespace cleanup", func(t *testing.T) {
		parentObj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "workspace.test/v1",
			"kind":       "Workspace",
			"metadata":   map[string]any{"namespace": tenantNS, "name": "w1"},
		}}
		dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(
			runtime.NewScheme(), parentListKinds(), parentObj, newChildObject(childNS, "bs-1"),
		)
		cs := k8sfake.NewClientset(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: childNS, Labels: ownerLabels},
		})

		writer := NewNamespaceManagingWriterAdapter[*testWorkspaceScopedIdentifiable](
			dynFake, cs, testParentGVR, logger, parentDomainToK8s, parentK8sToDomain,
			NoChildNamespace, []schema.GroupVersionResource{testChildGVR},
		)

		require.NoError(t, writer.Delete(context.Background(), parent))

		_, getErr := dynFake.Resource(testParentGVR).Namespace(tenantNS).Get(
			context.Background(), "w1", metav1.GetOptions{},
		)
		require.True(t, kerrs.IsNotFound(getErr))

		_, nsErr := cs.CoreV1().Namespaces().Get(context.Background(), childNS, metav1.GetOptions{})
		require.NoError(t, nsErr, "NoChildNamespace must not delete the child namespace")
	})
}

// Network CR lives in the workspace namespace; NetworkChildren provisions a separate
// per-network namespace for subnet/route-table (same shape as gateway wiring).
var testNetworkParentGVR = schema.GroupVersionResource{Group: "network.test", Version: "v1", Resource: "networks"}

func networkParentListKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{
		testNetworkParentGVR: "NetworkList",
		testChildGVR:         "BlockStorageList", // stand-in for subnet/route-table
	}
}

func networkParentDomainToK8s(m *testWorkspaceScopedIdentifiable) (client.Object, error) {
	// Network CRs sit in the workspace namespace (tenant + workspace), not the network child NS.
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "network.test/v1",
		"kind":       "Network",
		"metadata": map[string]any{
			"namespace": ComputeNamespace(&kernelresource.Scope{Tenant: m.tenant, Workspace: m.workspace}),
			"name":      m.name,
		},
	}}, nil
}

func networkParentK8sToDomain(obj client.Object) (*testWorkspaceScopedIdentifiable, error) {
	return &testWorkspaceScopedIdentifiable{name: obj.GetName(), tenant: "t1", workspace: "w1"}, nil
}

func TestNamespaceManagingWriterAdapter_Delete_NetworkChildren(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// NetworkChildren uses GetName() as the network segment and GetWorkspace() for the workspace.
	network := &testWorkspaceScopedIdentifiable{name: "net1", tenant: "t1", workspace: "w1"}
	workspaceNS := ComputeNamespace(&kernelresource.Scope{Tenant: "t1", Workspace: "w1"})
	networkChildNS := ComputeNetworkNamespace(fakeNetworkScope{tenant: "t1", workspace: "w1", network: "net1"})
	ownerLabels := map[string]string{
		labels.InternalTenantLabel:    "t1",
		labels.InternalWorkspaceLabel: "w1",
		labels.InternalNetworkLabel:   "net1",
	}

	t.Run("refuses delete when network child namespace has resources", func(t *testing.T) {
		parentObj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "network.test/v1",
			"kind":       "Network",
			"metadata":   map[string]any{"namespace": workspaceNS, "name": "net1"},
		}}
		dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(
			runtime.NewScheme(), networkParentListKinds(), parentObj, newChildObject(networkChildNS, "subnet-1"),
		)
		cs := k8sfake.NewClientset(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: networkChildNS, Labels: ownerLabels},
		})

		writer := NewNamespaceManagingWriterAdapter[*testWorkspaceScopedIdentifiable](
			dynFake, cs, testNetworkParentGVR, logger, networkParentDomainToK8s, networkParentK8sToDomain,
			NetworkChildren, []schema.GroupVersionResource{testChildGVR},
		)

		err := writer.Delete(context.Background(), network)
		require.Error(t, err)
		var domainErr *kernel.Error
		require.ErrorAs(t, err, &domainErr)
		require.Equal(t, kernel.KindConflict, domainErr.Kind)

		_, getErr := dynFake.Resource(testNetworkParentGVR).Namespace(workspaceNS).Get(
			context.Background(), "net1", metav1.GetOptions{},
		)
		require.NoError(t, getErr, "network CR must remain when children exist")

		_, nsErr := cs.CoreV1().Namespaces().Get(context.Background(), networkChildNS, metav1.GetOptions{})
		require.NoError(t, nsErr, "network child namespace must remain when children exist")
	})

	t.Run("deletes network and owned empty network child namespace", func(t *testing.T) {
		parentObj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "network.test/v1",
			"kind":       "Network",
			"metadata":   map[string]any{"namespace": workspaceNS, "name": "net1"},
		}}
		dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(
			runtime.NewScheme(), networkParentListKinds(), parentObj,
		)
		cs := k8sfake.NewClientset(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: networkChildNS, Labels: ownerLabels},
		})

		writer := NewNamespaceManagingWriterAdapter[*testWorkspaceScopedIdentifiable](
			dynFake, cs, testNetworkParentGVR, logger, networkParentDomainToK8s, networkParentK8sToDomain,
			NetworkChildren, []schema.GroupVersionResource{testChildGVR},
		)

		require.NoError(t, writer.Delete(context.Background(), network))

		_, getErr := dynFake.Resource(testNetworkParentGVR).Namespace(workspaceNS).Get(
			context.Background(), "net1", metav1.GetOptions{},
		)
		require.True(t, kerrs.IsNotFound(getErr), "network CR should be deleted")

		_, nsErr := cs.CoreV1().Namespaces().Get(context.Background(), networkChildNS, metav1.GetOptions{})
		require.True(t, kerrs.IsNotFound(nsErr), "owned empty network child namespace should be deleted")
	})
}

// testLabelled is a domain type carrying SECA labels the way the real ones do: the values live in
// metadata.labels under hashed kl/<sha3> keys, and the key list lives in commonData.labels. The
// key list is what KeyedToOriginal walks to rebuild them, so a write that drops commonData leaves
// a newly added key unreachable even though its value made it onto the object.
type testLabelled struct {
	name   string
	labels map[string]string
}

func (t *testLabelled) GetName() string      { return t.name }
func (t *testLabelled) GetVersion() string   { return "" }
func (t *testLabelled) GetTenant() string    { return "t1" }
func (t *testLabelled) GetWorkspace() string { return "w1" }

func testLabelledToCR(d *testLabelled) (client.Object, error) {
	crLabels := map[string]any{}
	for k, v := range labels.OriginalToKeyed(d.labels) {
		crLabels[k] = v
	}

	keys := make([]any, 0, len(d.labels))
	for _, k := range slices.Sorted(maps.Keys(d.labels)) {
		keys = append(keys, k)
	}

	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "network.test/v1",
		"kind":       "RouteTable",
		"metadata": map[string]any{
			"namespace": ComputeNamespace(&kernelresource.Scope{Tenant: "t1", Workspace: "w1"}),
			"name":      d.name,
			"labels":    crLabels,
		},
		"commonData": map[string]any{"labels": keys},
		"spec":       map[string]any{},
	}}, nil
}

func testLabelledFromCR(obj client.Object) (*testLabelled, error) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", obj)
	}

	keys, _, err := unstructured.NestedStringSlice(u.Object, "commonData", "labels")
	if err != nil {
		return nil, err
	}

	return &testLabelled{name: u.GetName(), labels: labels.KeyedToOriginal(labels.GetKeyedLabels(u.GetLabels()), keys)}, nil
}

// TestWriterAdapter_Update_PropagatesCommonData is a regression test for a label added on update
// going missing. commonData is a sibling of spec, not part of it, so an update that copied only
// spec, labels and annotations wrote the new label's value but never its key - and the key list is
// what the read path uses to find it again.
func TestWriterAdapter_Update_PropagatesCommonData(t *testing.T) {
	namespace := ComputeNamespace(&kernelresource.Scope{Tenant: "t1", Workspace: "w1"})

	created, err := testLabelledToCR(&testLabelled{name: "rt-1", labels: map[string]string{"env": "prod"}})
	require.NoError(t, err)

	dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(), testListKinds(), created.(*unstructured.Unstructured))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	writer := NewWriterAdapter[*testLabelled](dynFake, testGVR, logger, testLabelledToCR, testLabelledFromCR)

	// Add a second label, the case that used to be lost.
	updated, err := writer.Update(context.Background(), &testLabelled{
		name:   "rt-1",
		labels: map[string]string{"env": "prod", "tier": "frontend"},
	})
	require.NoError(t, err)
	require.Equal(t, map[string]string{"env": "prod", "tier": "frontend"}, (*updated).labels)

	stored, err := dynFake.Resource(testGVR).Namespace(namespace).
		Get(context.Background(), "rt-1", metav1.GetOptions{})
	require.NoError(t, err)

	keys, _, err := unstructured.NestedStringSlice(stored.Object, "commonData", "labels")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"env", "tier"}, keys, "the added label's key must reach commonData")
}

// TestWriterAdapter_Update_NoOpDoesNotWrite pins the early return in updateMetadataAndSpecRetry: an
// update whose desired state already matches what is stored must not issue a write.
//
// It is not an optimisation. A write bumps resourceVersion, the controller watches its own writes,
// and an active resource's reconcile hands the plugin a level-triggered Update - so a PUT that
// changes nothing still costs a reconcile and a round trip to the provider. commonData is the part
// worth guarding: it carries the label *key list*, which converters build from a Go map, and an
// equal-but-reordered list compares unequal here and writes. That is why they sort it.
func TestWriterAdapter_Update_NoOpDoesNotWrite(t *testing.T) {
	labelled := &testLabelled{name: "rt-1", labels: map[string]string{"env": "prod", "tier": "frontend", "team": "platform"}}

	created, err := testLabelledToCR(labelled)
	require.NoError(t, err)

	dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(), testListKinds(), created.(*unstructured.Unstructured))

	var writes int
	dynFake.PrependReactor("update", "*", func(k8stesting.Action) (bool, runtime.Object, error) {
		writes++
		return false, nil, nil // Count it, then fall through to the tracker.
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	writer := NewWriterAdapter[*testLabelled](dynFake, testGVR, logger, testLabelledToCR, testLabelledFromCR)

	// Repeated, because an unstable key ordering would only collide with the stored order some of
	// the time - one pass could miss it by luck, ten will not.
	for i := range 10 {
		_, err := writer.Update(context.Background(), labelled)
		require.NoErrorf(t, err, "no-op update %d should succeed", i)
	}

	require.Zerof(t, writes, "an update that changes nothing must not write, got %d writes", writes)
}
