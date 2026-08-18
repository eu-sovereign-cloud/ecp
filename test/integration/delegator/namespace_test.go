//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

// TestWorkspaceNamespaceLifecycle covers the namespace a Workspace owns for its children,
// end to end and with nothing hand-created: the write path provisions it on create, the
// gateway refuses to delete a Workspace whose namespace still holds resources, and the
// delegator's controller finalizer tears it down once the CR is really gone.
func TestWorkspaceNamespaceLifecycle(t *testing.T) {
	wsName := "ns-ws-" + uuid.New().String()[:8]
	childNS := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant, Workspace: wsName})

	newWorkspace := func() *wsdom.Workspace {
		return &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: wsName},
				Scope:          resource.Scope{Tenant: testTenant},
			},
		}
	}
	newBlockStorage := func(name string) *bsdom.BlockStorage {
		return &bsdom.BlockStorage{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: name},
				Scope:          resource.Scope{Tenant: testTenant, Workspace: wsName},
			},
			Spec: bsdom.BlockStorageSpec{
				SizeGB: 1,
				SkuRef: commondomain.Reference{Region: testRegion, Resource: "sku-1"},
			},
		}
	}

	// A failure part-way through must not strand the namespace for every later run.
	t.Cleanup(func() {
		_ = workspaceRepo.Delete(context.Background(), newWorkspace())
	})

	_, err := workspaceRepo.Create(t.Context(), newWorkspace())
	require.NoError(t, err, "workspace should be created")

	// Create: the child namespace exists, labelled with who provisioned it. Those labels
	// are what the controller later checks before it deletes anything.
	requireNamespaceLabels(t, childNS, map[string]string{
		labels.InternalTenantLabel:    testTenant,
		labels.InternalWorkspaceLabel: wsName,
	})

	bsName := "ns-bs-" + uuid.New().String()[:8]
	_, err = blockStorageRepo.Create(t.Context(), newBlockStorage(bsName))
	require.NoError(t, err, "block storage should be created in the workspace namespace")

	// Refusal: a workspace whose namespace still holds a child cannot be deleted. Without
	// this the namespace delete would cascade and take the block storage with it.
	err = workspaceRepo.Delete(t.Context(), newWorkspace())
	require.ErrorIs(t, err, kernel.ErrConflict, "deleting a non-empty workspace should be refused")
	requireNamespaceExists(t, childNS)

	require.NoError(t, blockStorageRepo.Delete(t.Context(), newBlockStorage(bsName)))
	require.NoError(t, waitGone(t.Context(), blockStorageRepo, newBlockStorage(bsName)))

	// Delete: with the namespace empty the workspace goes, and its controller reclaims the
	// namespace behind it.
	require.NoError(t, workspaceRepo.Delete(t.Context(), newWorkspace()))
	require.NoError(t, waitGone(t.Context(), workspaceRepo, newWorkspace()))
	requireNamespaceGone(t, childNS)

	// Drift: no namespace anywhere still claims to belong to this workspace, and the tenant
	// namespace — which nothing cascades from — is still standing.
	requireNoNamespacesLabelled(t, labels.InternalWorkspaceLabel+"="+wsName)
	requireNamespaceExists(t, k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant}))
}

// TestNetworkNamespaceLifecycle is the same contract one level down the scope chain: a
// Network owns the namespace its network-scoped children (Subnet, RouteTable) live in.
func TestNetworkNamespaceLifecycle(t *testing.T) {
	netName := "ns-net-" + uuid.New().String()[:8]

	newNetwork := func() *netdom.Network {
		return &netdom.Network{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: netName},
				Scope:          resource.Scope{Tenant: testTenant, Workspace: testWorkspace},
			},
			Spec: netdom.NetworkSpec{
				CIDR:   netdom.CIDR{IPv4: networkCIDR},
				SkuRef: commondomain.Reference{Resource: "sku-1"},
			},
		}
	}
	newSubnet := func(name string) *subnetdom.Subnet {
		s := &subnetdom.Subnet{
			RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{Name: name},
				},
				Network: netName,
			},
			Spec: subnetdom.SubnetSpec{
				Cidr:          subnetdom.CIDR{IPv4: "10.30.1.0/24"},
				RouteTableRef: commondomain.Reference{Resource: "route-tables/rt1"},
				Zone:          "zone-1",
			},
		}
		s.Tenant = testTenant
		s.Workspace = testWorkspace
		return s
	}

	// The hash is over tenant/workspace/network, so any network-scoped identity under this
	// network names the same namespace — the subnet's own name is irrelevant here.
	childNS := k8sadapter.ComputeNetworkNamespace(newSubnet("probe"))

	t.Cleanup(func() {
		_ = networkRepo.Delete(context.Background(), newNetwork())
	})

	_, err := networkRepo.Create(t.Context(), newNetwork())
	require.NoError(t, err, "network should be created")

	requireNamespaceLabels(t, childNS, map[string]string{
		labels.InternalTenantLabel:    testTenant,
		labels.InternalWorkspaceLabel: testWorkspace,
		labels.InternalNetworkLabel:   netName,
	})

	subnetName := "ns-subnet-" + uuid.New().String()[:8]
	_, err = subnetRepo.Create(t.Context(), newSubnet(subnetName))
	require.NoError(t, err, "subnet should be created in the network namespace")

	err = networkRepo.Delete(t.Context(), newNetwork())
	require.ErrorIs(t, err, kernel.ErrConflict, "deleting a network with a subnet should be refused")
	requireNamespaceExists(t, childNS)

	require.NoError(t, subnetRepo.Delete(t.Context(), newSubnet(subnetName)))
	require.NoError(t, waitGone(t.Context(), subnetRepo, newSubnet(subnetName)))

	require.NoError(t, networkRepo.Delete(t.Context(), newNetwork()))
	require.NoError(t, waitGone(t.Context(), networkRepo, newNetwork()))
	requireNamespaceGone(t, childNS)

	requireNoNamespacesLabelled(t, labels.InternalNetworkLabel+"="+netName)
	// The workspace namespace the network lived in is not the network's to reclaim.
	requireNamespaceExists(t, k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant, Workspace: testWorkspace}))
}

// TestNamespaceOwnerLabelDrift covers the convergence half of the create path: a namespace
// that exists but is missing its owner labels gets them stamped on the next write. That is
// not hypothetical — deploy.sh creates the tenant namespace bare, because the catalog
// fixtures are applied before any request reaches the gateway, and a child namespace that
// stayed unlabelled would be one its own controller refuses to delete.
func TestNamespaceOwnerLabelDrift(t *testing.T) {
	tenantNS := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant})
	workspaceNS := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant, Workspace: testWorkspace})

	t.Run("tenant namespace is adopted by the first tenant-scoped write", func(t *testing.T) {
		// Nothing to arrange: deploy.sh created this one bare and TestMain's workspace
		// create is the first tenant-scoped write against it.
		requireNamespaceLabels(t, tenantNS, map[string]string{labels.InternalTenantLabel: testTenant})
	})

	t.Run("fixture namespaces are provisioned by the write path, not by hand", func(t *testing.T) {
		// TestMain hand-created neither of these; they exist because the workspace and
		// network fixtures were written through namespace-managing adapters, and the owner
		// labels are the proof — a hand-made namespace would have none.
		requireNamespaceLabels(t, workspaceNS, map[string]string{
			labels.InternalTenantLabel:    testTenant,
			labels.InternalWorkspaceLabel: testWorkspace,
		})
		requireNamespaceLabels(t, testNetworkNamespace(), map[string]string{
			labels.InternalTenantLabel:    testTenant,
			labels.InternalWorkspaceLabel: testWorkspace,
			labels.InternalNetworkLabel:   testNetwork,
		})
	})

	// Repair belongs to the delegator, not the write path: the gateway only touches the child
	// namespace after a CR write it actually performed, so a create that fails with
	// AlreadyExists no longer stamps anything. Every reconcile of the owning CR re-runs
	// NamespaceEnsure instead, which is a stronger guarantee — it does not need anyone to
	// re-issue a create — but an asynchronous one, so this polls.
	t.Run("stripped owner labels are restored by the owning controller", func(t *testing.T) {
		stripOwnerLabels(t, workspaceNS)

		// A settled resource is only resynced on the manager's default period, so nudge the CR
		// to get a reconcile now rather than waiting on it.
		touchWorkspace(t, testWorkspace)
		requireNamespaceLabelsEventually(t, workspaceNS, map[string]string{
			labels.InternalTenantLabel:    testTenant,
			labels.InternalWorkspaceLabel: testWorkspace,
		})
	})
}

// touchWorkspace annotates the Workspace CR so the controller reconciles it now. The value is
// unique per call: an identical patch is a no-op that generates no watch event.
func touchWorkspace(t *testing.T, name string) {
	t.Helper()
	patch, err := json.Marshal(map[string]any{"metadata": map[string]any{"annotations": map[string]any{
		"test.secapi.cloud/touch": uuid.New().String(),
	}}})
	require.NoError(t, err)

	_, err = dynamicClient.Resource(wsk8s.WorkspaceGVR).
		Namespace(k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant})).
		Patch(t.Context(), name, types.MergePatchType, patch, metav1.PatchOptions{})
	require.NoError(t, err)
}

// requireNamespaceLabelsEventually is requireNamespaceLabels for a repair that happens on a
// reconcile rather than in the request the test just made.
func requireNamespaceLabelsEventually(t *testing.T, name string, expected map[string]string) {
	t.Helper()
	err := wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		ns, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		for k, v := range expected {
			if ns.Labels[k] != v {
				return false, nil
			}
		}
		return true, nil
	})
	require.NoErrorf(t, err, "namespace %q should have its owner labels restored by the controller", name)
}

// requireNamespaceLabels asserts the namespace exists and carries every expected label.
// Other labels are allowed: the check mirrors namespaceOwnedBy, which is a subset test.
func requireNamespaceLabels(t *testing.T, name string, expected map[string]string) {
	t.Helper()
	ns, err := clientset.CoreV1().Namespaces().Get(t.Context(), name, metav1.GetOptions{})
	require.NoErrorf(t, err, "namespace %q should exist", name)
	for k, v := range expected {
		require.Equalf(t, v, ns.Labels[k], "namespace %q should carry owner label %s", name, k)
	}
}

func requireNamespaceExists(t *testing.T, name string) {
	t.Helper()
	_, err := clientset.CoreV1().Namespaces().Get(t.Context(), name, metav1.GetOptions{})
	require.NoErrorf(t, err, "namespace %q should still exist", name)
}

// requireNamespaceGone polls until the namespace is fully gone. Teardown is asynchronous —
// the controller deletes it from its finalizer, and Kubernetes then drains it — so a single
// Get would be a race.
func requireNamespaceGone(t *testing.T, name string) {
	t.Helper()
	err := wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		_, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
		return kerrs.IsNotFound(err), nil
	})
	require.NoErrorf(t, err, "namespace %q should have been reclaimed by the owning controller", name)
}

// requireNoNamespacesLabelled asserts no namespace matches the selector, catching a
// namespace leaked under a hash the test did not predict.
func requireNoNamespacesLabelled(t *testing.T, selector string) {
	t.Helper()
	list, err := clientset.CoreV1().Namespaces().List(t.Context(), metav1.ListOptions{LabelSelector: selector})
	require.NoError(t, err)
	require.Emptyf(t, list.Items, "no namespace should be left matching %q", selector)
}

// stripOwnerLabels removes the owner labels from a namespace, simulating one provisioned
// before the labels existed (a hand-applied fixture, an older release).
func stripOwnerLabels(t *testing.T, name string) {
	t.Helper()
	patch, err := json.Marshal(map[string]any{"metadata": map[string]any{"labels": map[string]any{
		labels.InternalTenantLabel:    nil,
		labels.InternalWorkspaceLabel: nil,
		labels.InternalNetworkLabel:   nil,
	}}})
	require.NoError(t, err)

	_, err = clientset.CoreV1().Namespaces().Patch(t.Context(), name, types.MergePatchType, patch, metav1.PatchOptions{})
	require.NoError(t, err)
}
