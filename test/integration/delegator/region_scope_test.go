//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
)

// foreignRegion is a region NO delegator in the test stack serves. It has to be a third
// region: the stack deploys one delegator per region it serves (itbg-bergamo and
// region-two — internal/deploy/delegator*/values.yaml), so naming either of those would
// assert that a delegator which is deployed for a region ignores it.
//
// It is deliberately not in test-data/regions.yaml either. Nothing here goes through a
// gateway — the CR is written straight through a repo adapter — so the region only has to
// be a legal label value, and a catalog entry advertising a region no gateway serves would
// be the more misleading fixture.
const foreignRegion = "region-unserved"

// TestRegionScopedDelegator is the negative half of the region scope the whole suite runs
// against: this delegator is deployed for testRegion only (internal/deploy/delegator/values.yaml)
// and its sibling for region-two, so a CR in a region neither serves must be left strictly
// alone — no finalizer, no status, nothing. Reconciling it anyway is how one region's
// delegator ends up provisioning into another's backend, which no later check would catch.
// The positive half — that the sibling does pick up its own region — is
// test/e2e/region_isolation_test.go, which needs the whole stack.
//
// The in-region resource is the control. Without it the assertion would also pass against a
// delegator that had crashed, which is the failure this test is most likely to be masked by.
func TestRegionScopedDelegator(t *testing.T) {
	suffix := uuid.New().String()[:8]
	foreign := newRegionalBlockStorage("test-bs-foreign-"+suffix, foreignRegion)
	control := newRegionalBlockStorage("test-bs-in-region-"+suffix, testRegion)

	_, err := blockStorageRepo.Create(t.Context(), foreign)
	require.NoError(t, err, "a resource in another region is still a valid write")
	t.Cleanup(func() {
		// Nothing holds a finalizer on it, so this is immediate — and it has to happen
		// before the suite tears the workspace down, which is refused while its namespace
		// still holds a block storage.
		require.NoError(t, blockStorageRepo.Delete(context.WithoutCancel(t.Context()), foreign))
	})

	_, err = blockStorageRepo.Create(t.Context(), control)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, blockStorageRepo.Delete(context.WithoutCancel(t.Context()), control))
	})

	require.NoError(t, waitForBlockStorageActive(t.Context(), blockStorageRepo, control.Name),
		"the in-region control must reconcile, otherwise the delegator is simply down")

	// The finalizer is the sharpest signal: the controller adds it on the very first
	// reconcile, before anything else, so its absence means this CR never reached one.
	var cr bsk8s.BlockStorage
	key := types.NamespacedName{
		Name:      foreign.Name,
		Namespace: k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant, Workspace: testWorkspace}),
	}
	require.NoError(t, k8sClient.Get(t.Context(), key, &cr))
	require.Empty(t, cr.GetFinalizers(), "an out-of-region CR must never be reconciled")
	require.Nil(t, cr.Status, "an out-of-region CR must never be given a status")
}

// newRegionalBlockStorage builds a leaf resource in the suite's workspace, in the region given.
// Block storage is the simplest one to place: it owns no namespace and has no children.
func newRegionalBlockStorage(name, region string) *bsdom.BlockStorage {
	return &bsdom.BlockStorage{
		RegionalMetadata: commondomain.RegionalMetadata{
			Region:         region,
			CommonMetadata: commondomain.CommonMetadata{Name: name},
			Scope:          resource.Scope{Tenant: testTenant, Workspace: testWorkspace},
		},
		Spec: bsdom.BlockStorageSpec{
			SizeGB: 1,
			SkuRef: commondomain.Reference{Region: region, Resource: "sku-1"},
		},
	}
}
