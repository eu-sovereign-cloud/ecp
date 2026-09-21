//go:build e2e

package e2e

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	storagev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	workspacev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	authhelper "github.com/eu-sovereign-cloud/ecp/test/internal/authhelper"
	"github.com/eu-sovereign-cloud/ecp/test/internal/testenv"
)

// The two delegator releases the test stack deploys, one per region, into the same
// cluster: ecp-delegator serves testRegion and ecp-delegator-two serves secondRegion
// (internal/deploy/delegator/values.yaml and internal/deploy/delegator-two/values.yaml).
// The chart gives both pods the same `app: delegator` label, so the release — which the
// chart puts on every object as app.kubernetes.io/instance — is what tells them apart.
const (
	delegatorOneSelector = "app.kubernetes.io/instance=ecp-delegator"
	delegatorTwoSelector = "app.kubernetes.io/instance=ecp-delegator-two"
)

// TestRegionIsolationEndToEnd is the multi-region stack driven the way a tenant drives it:
// one gateway deployment serving two regions, and one delegator per region sharing the
// cluster. TestMultiRegionEndToEnd already covers that a second region reconciles at all;
// what this adds is everything that only shows up once the regions can collide —
//
//   - the same workspace name created in both regions is two workspaces, each in its own
//     per-region namespace, and each reconciled by the delegator deployed for that region
//     and by that one alone;
//   - a GET or a list in one region does not surface the other region's resources;
//   - a delete in one region takes nothing in the other region with it.
//
// The REST half of this is asserted without a delegator in
// test/integration/gateway-regional/multiregion_test.go. Here the point is the rest of the
// chain: which delegator woke up, where the CR was actually placed, and what survives a
// delete — none of which the gateway suites can see.
func TestRegionIsolationEndToEnd(t *testing.T) {
	ctx := context.Background()

	// One name, created in both regions. Sharing the name is the whole point: it is what
	// would collide if a workspace were still keyed by tenant alone.
	wsName := "e2e-ri-ws-" + uuid.New().String()[:8]
	// A block storage in the second region only. It is the discriminator for "which
	// delegator did the work": both regions hold a workspace of the same name, so the
	// workspace name appears in both delegators' logs and cannot tell them apart.
	bsName := "e2e-ri-bs-" + uuid.New().String()[:8]

	homeWS := workspaceClient                           // testRegion, no prefix — the gateway's default region
	awayWS := regionWorkspaceClient(t, secondRegion)    // secondRegion, under its /regions/<region> prefix
	awayStorage := regionStorageClient(t, secondRegion) // the same, for seca.storage
	homeStorage := storageClient                        // testRegion, no prefix
	regions := map[string]*workspacev1.ClientWithResponses{testRegion: homeWS, secondRegion: awayWS}

	t.Cleanup(func() {
		// Innermost first: a workspace delete is refused with 409 while anything still
		// lives in the namespace it owns for its children — and that namespace is shared
		// by both regions' copies, so the block storage blocks BOTH deletes, not just its
		// own region's.
		testenv.DeleteUntilGone(ctx, func() (*http.Response, error) {
			return awayStorage.DeleteBlockStorage(ctx, testTenant, wsName, bsName, nil)
		})
		for _, c := range regions {
			testenv.DeleteUntilGone(ctx, func() (*http.Response, error) {
				return c.DeleteWorkspace(ctx, testTenant, wsName, nil)
			})
		}
	})

	// Step 1: the same name in both regions is two workspaces, in two namespaces, each
	// reconciled to Active by the delegator that serves its region.
	t.Run("the same workspace name in two regions is two workspaces in two namespaces", func(t *testing.T) {
		for region, client := range regions {
			// The spec says which region wrote it, so a GET that resolved the wrong
			// region's CR is visible rather than merely indistinguishable.
			resp, err := client.CreateOrUpdateWorkspaceWithResponse(ctx, testTenant, wsName, nil,
				schema.Workspace{Spec: map[string]any{"where": region}})
			require.NoError(t, err)
			require.Equalf(t, http.StatusOK, resp.StatusCode(),
				"creating %q in %q must not collide with the other region", wsName, region)
			require.NotNil(t, resp.JSON200)
			require.NotNil(t, resp.JSON200.Metadata)
			require.Equal(t, region, resp.JSON200.Metadata.Region)

			waitForActive(t, "workspace "+wsName+" in "+region, func(ctx context.Context) (schema.ResourceState, bool, error) {
				r, err := client.GetWorkspaceWithResponse(ctx, testTenant, wsName)
				if err != nil {
					return "", false, err
				}
				if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
					return "", false, nil
				}
				return r.JSON200.Status.State, true, nil
			})
		}

		// Both reached Active, so both delegators are alive and each accepted its own
		// region — the workspace controller is region-scoped on both deployments.

		// And the two CRs really are two objects in two namespaces. This is the
		// ComputeRegionNamespace formula seen from the cluster: same tenant, same name,
		// different region, so nothing about the placement is shared.
		homeNS := workspaceNamespace(testRegion)
		awayNS := workspaceNamespace(secondRegion)
		require.NotEqual(t, homeNS, awayNS, "the two regions must not resolve to one namespace")
		for region, ns := range map[string]string{testRegion: homeNS, secondRegion: awayNS} {
			_, err := k8s.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
			require.NoErrorf(t, err, "the per-region namespace for %q (%s) must exist", region, ns)
		}
	})

	// Step 2: a block storage created through the second region is reconciled by the
	// delegator deployed for that region — and by no other.
	t.Run("only the delegator serving the region reconciles that region's resources", func(t *testing.T) {
		body := schema.BlockStorage{
			Spec: schema.BlockStorageSpec{SizeGB: 1, SkuRef: schema.Reference{Resource: "sku-1"}},
		}
		resp, err := awayStorage.CreateOrUpdateBlockStorageWithResponse(ctx, testTenant, wsName, bsName, nil, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.NotNil(t, resp.JSON200)
		require.Equal(t, secondRegion, resp.JSON200.Metadata.Region)

		waitForActive(t, "block storage "+bsName, func(ctx context.Context) (schema.ResourceState, bool, error) {
			r, err := awayStorage.GetBlockStorageWithResponse(ctx, testTenant, wsName, bsName)
			if err != nil {
				return "", false, err
			}
			if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
				return "", false, nil
			}
			return r.JSON200.Status.State, true, nil
		})

		// Reaching Active proves *a* delegator reconciled it; the logs are what say which.
		// The dummy plugin logs every operation it performs with the resource name, and
		// the region scope is a watch predicate, so a delegator that was not meant to see
		// this CR never entered Reconcile and never logged the name. Both CRs are in the
		// same namespace and carry the same fields, so nothing on the CR itself could
		// distinguish the two delegators.
		require.Contains(t, delegatorLogs(t, ctx, delegatorTwoSelector), bsName,
			"the delegator deployed for %s must have reconciled it", secondRegion)
		require.NotContains(t, delegatorLogs(t, ctx, delegatorOneSelector), bsName,
			"the delegator deployed for %s must never have seen another region's CR", testRegion)
	})

	// Step 3 (retrieve): what a read in one region is allowed to see of another's.
	t.Run("a read in one region does not return another region's resources", func(t *testing.T) {
		// A workspace is keyed by tenant AND region, so one that exists in only one
		// region is simply absent from the other — a 404, not another region's copy.
		onlyAway := "e2e-ri-away-" + uuid.New().String()[:8]
		t.Cleanup(func() {
			testenv.DeleteUntilGone(ctx, func() (*http.Response, error) {
				return awayWS.DeleteWorkspace(ctx, testTenant, onlyAway, nil)
			})
		})
		created, err := awayWS.CreateOrUpdateWorkspaceWithResponse(ctx, testTenant, onlyAway, nil, schema.Workspace{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, created.StatusCode())

		missing, err := homeWS.GetWorkspaceWithResponse(ctx, testTenant, onlyAway)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, missing.StatusCode(),
			"a workspace created in %s must not be addressable in %s", secondRegion, testRegion)

		// And where the name DOES exist in both, each region resolves its own.
		for region, client := range regions {
			got, err := client.GetWorkspaceWithResponse(ctx, testTenant, wsName)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, got.StatusCode())
			require.Equal(t, region, got.JSON200.Metadata.Region)
			require.Equalf(t, region, got.JSON200.Spec["where"],
				"a GET in %q resolved the other region's workspace", region)
		}

		// Below a workspace the namespace formula has no region dimension, so the region
		// label on the CR is the only thing separating the two regions' resources — and it
		// is enforced on lists, which is where a tenant enumerates what a region holds.
		require.Contains(t, blockStorageNames(t, ctx, awayStorage, wsName), bsName)
		require.NotContains(t, blockStorageNames(t, ctx, homeStorage, wsName), bsName,
			"a list in %s must not return a resource created in %s", testRegion, secondRegion)

		// Deliberately not asserted: a GET of that same block storage through the
		// testRegion prefix. Item operations below a workspace are region-blind by design
		// — both regions' resources share sha3-224(tenant/workspace) and a GET addresses
		// the CR by name — so requiring a 404 here would be requiring the opposite of what
		// doc/ARCHITECTURE.md ("Multi-region gateways") specifies.
	})

	// Step 4 (delete): a delete is confined to the region it was addressed to.
	t.Run("a delete in one region leaves the other region standing", func(t *testing.T) {
		// While the block storage is up, NEITHER region may delete the workspace: the 409
		// counts what lives in the children namespace, which both regions' copies share.
		for region, client := range regions {
			resp, err := client.DeleteWorkspaceWithResponse(ctx, testTenant, wsName, nil)
			require.NoError(t, err)
			require.Equalf(t, http.StatusConflict, resp.StatusCode(),
				"deleting %q in %q must be refused while the shared children namespace is not empty", wsName, region)
		}

		testenv.DeleteUntilGone(ctx, func() (*http.Response, error) {
			return awayStorage.DeleteBlockStorage(ctx, testTenant, wsName, bsName, nil)
		})

		// Now delete the second region's workspace only. The API accepts the delete and a
		// finalizer clears the CR afterwards, so wait for it to actually be gone: until it
		// is, the cleanup hook that has to notice the co-owner has not run yet.
		testenv.DeleteUntilGone(ctx, func() (*http.Response, error) {
			return awayWS.DeleteWorkspace(ctx, testTenant, wsName, nil)
		})
		waitForGone(t, "workspace "+wsName+" in "+secondRegion, func(ctx context.Context) (int, error) {
			r, err := awayWS.GetWorkspaceWithResponse(ctx, testTenant, wsName)
			if err != nil {
				return 0, err
			}
			return r.StatusCode(), nil
		})

		// The first region's workspace of the same name is untouched, and still Active:
		// nothing re-reconciled it into an error because its namespace went away.
		survivor, err := homeWS.GetWorkspaceWithResponse(ctx, testTenant, wsName)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, survivor.StatusCode(),
			"deleting a workspace in %s must not delete the same name in %s", secondRegion, testRegion)
		require.NotNil(t, survivor.JSON200.Status)
		require.Equal(t, schema.ResourceStateActive, survivor.JSON200.Status.State)
		require.Equal(t, testRegion, survivor.JSON200.Spec["where"])

		// And the namespace the two copies shared for their children is still there. Both
		// owned it, so the delegator that tore the second region's workspace down had to
		// find the live co-owner and defer to it; reclaiming it here would strand the
		// survivor's children until its next resync.
		shared, err := k8s.CoreV1().Namespaces().Get(ctx, childNamespace(wsName), metav1.GetOptions{})
		require.NoError(t, err,
			"the children namespace is co-owned, so it goes with the last of the two workspaces, not the first")
		// Still Active, not Terminating: a cleanup that deleted it anyway would leave the
		// namespace readable for a while yet, so its mere existence is not the assertion.
		require.Equal(t, corev1.NamespaceActive, shared.Status.Phase)
	})
}

// waitForGone polls get until it reports 404, failing the test if that does not happen
// within activeTimeout. A delete is accepted synchronously but completes when the
// controller's finalizer has run, so anything asserting on what the delete left behind has
// to wait for the CR to disappear first.
func waitForGone(t *testing.T, what string, get func(context.Context) (int, error)) {
	t.Helper()
	err := wait.PollUntilContextTimeout(context.Background(), activePollInterval, activeTimeout, true,
		func(ctx context.Context) (bool, error) {
			status, err := get(ctx)
			if err != nil {
				return false, err
			}
			return status == http.StatusNotFound, nil
		})
	require.NoErrorf(t, err, "%s was still there after %s", what, activeTimeout)
}

// regionWorkspaceClient returns a workspace client rooted at one region's path prefix —
// the base URL that region's entry in the catalog advertises.
func regionWorkspaceClient(t *testing.T, region string) *workspacev1.ClientWithResponses {
	t.Helper()
	c, err := workspacev1.NewClientWithResponses(
		regionalURL+"/regions/"+region+"/providers/seca.workspace",
		workspacev1.WithRequestEditorFn(authhelper.AdminEditor()))
	require.NoError(t, err)
	return c
}

// regionStorageClient is regionWorkspaceClient for the seca.storage provider.
func regionStorageClient(t *testing.T, region string) *storagev1.ClientWithResponses {
	t.Helper()
	c, err := storagev1.NewClientWithResponses(
		regionalURL+"/regions/"+region+"/providers/seca.storage",
		storagev1.WithRequestEditorFn(authhelper.AdminEditor()))
	require.NoError(t, err)
	return c
}

// blockStorageNames returns every block storage name the client sees in one workspace.
func blockStorageNames(t *testing.T, ctx context.Context, c *storagev1.ClientWithResponses, workspace string) []string {
	t.Helper()
	resp, err := c.ListBlockStoragesWithResponse(ctx, testTenant, workspace, &storagev1.ListBlockStoragesParams{})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotNil(t, resp.JSON200)

	names := make([]string, 0, len(resp.JSON200.Items))
	for _, item := range resp.JSON200.Items {
		if item.Metadata != nil {
			names = append(names, item.Metadata.Name)
		}
	}
	return names
}

// workspaceNamespace is the namespace a workspace of testTenant lives in in one region,
// computed by the product's own formula rather than restated here — a test that hashed it
// itself would keep passing if the formula changed under it.
func workspaceNamespace(region string) string {
	return k8sadapter.ComputeRegionNamespace(&wsdom.Workspace{
		RegionalMetadata: commondomain.RegionalMetadata{
			Region: region,
			Scope:  resource.Scope{Tenant: testTenant},
		},
	})
}

// childNamespace is the namespace a workspace owns for its children. It has no region
// dimension — that is exactly why two same-named workspaces in two regions share one.
func childNamespace(workspace string) string {
	return k8sadapter.ComputeNamespace(&resource.Scope{Tenant: testTenant, Workspace: workspace})
}

// delegatorLogs returns the concatenated logs of every pod of one delegator release.
// Reading them is the only way to attribute a reconcile to a deployment: the region scope
// is a watch predicate, so a delegator outside the scope leaves no trace on the CR — which
// is the point, and also what makes the CR useless as evidence.
func delegatorLogs(t *testing.T, ctx context.Context, selector string) string {
	t.Helper()
	pods, err := k8s.CoreV1().Pods(systemNamespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	require.NoError(t, err)
	require.NotEmptyf(t, pods.Items, "no delegator pod matches %q — the stack is not deployed as this suite expects", selector)

	var all strings.Builder
	for _, pod := range pods.Items {
		stream, err := k8s.CoreV1().Pods(systemNamespace).
			GetLogs(pod.Name, &corev1.PodLogOptions{}).Stream(ctx)
		if apierrors.IsNotFound(err) {
			continue // the pod went away between the List and the GetLogs
		}
		require.NoError(t, err)
		body, err := io.ReadAll(stream)
		_ = stream.Close()
		require.NoError(t, err)
		all.Write(body)
	}
	return all.String()
}
