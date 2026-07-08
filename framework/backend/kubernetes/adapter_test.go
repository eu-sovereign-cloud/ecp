package kubernetes

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	scope := fakeNetworkScope{tenant: "t1", workspace: "w1", network: "n1"}

	require.Equal(t, ComputeNetworkNamespace(scope), ComputeNetworkNamespace(scope))
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
