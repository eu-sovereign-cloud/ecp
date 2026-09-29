//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	dummyplugin "github.com/eu-sovereign-cloud/ecp/csp/dummy/pkg/plugin"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	resource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/test/internal/testenv"
)

func TestWorkspace(t *testing.T) {
	// t.Parallel()
	t.Run("should create a workspace resource", func(t *testing.T) {
		// t.Parallel()

		//
		// Given a unique workspace domain resource definition
		workspaceName := "test-ws-create-" + uuid.New().String()[:8]
		wsDomain := &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				Region: testRegion,
				CommonMetadata: commondomain.CommonMetadata{
					Name: workspaceName,
				},
				Scope: resource.Scope{
					Tenant: testTenant,
				},
			},
		}

		//
		// When we create the workspace resource via the adapter
		_, err := workspaceRepo.Create(t.Context(), wsDomain)
		require.NoError(t, err)

		//
		// Then the resource should eventually become active
		var loadedWs *wsdom.Workspace
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedWs = &wsdom.Workspace{
				RegionalMetadata: commondomain.RegionalMetadata{
					Region: testRegion,
					CommonMetadata: commondomain.CommonMetadata{
						Name: workspaceName,
					},
					Scope: resource.Scope{
						Tenant: testTenant,
					},
				},
			}
			if err := workspaceRepo.Load(ctx, &loadedWs); err != nil {
				return false, err
			}
			if loadedWs.Status != nil && loadedWs.Status.State == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "workspace resource should become active")
		require.NotNil(t, loadedWs)
		require.NotNil(t, loadedWs.Status)
		require.NotNil(t, loadedWs.Status.State)
		require.Equal(t, commondomain.ResourceStateActive, loadedWs.Status.State)
		//
		// And we can cleanup the workspace
		err = workspaceRepo.Delete(t.Context(), wsDomain)
		require.NoError(t, err)
	})

	t.Run("should delete a workspace resource", func(t *testing.T) {
		// t.Parallel()

		//
		// Given a unique workspace resource that is already created
		workspaceName := "test-ws-delete-" + uuid.New().String()[:8]
		wsDomain := &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				Region: testRegion,
				CommonMetadata: commondomain.CommonMetadata{
					Name: workspaceName,
				},
				Scope: resource.Scope{
					Tenant: testTenant,
				},
			},
		}
		_, err := workspaceRepo.Create(t.Context(), wsDomain)
		require.NoError(t, err)

		var loadedWs *wsdom.Workspace
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedWs = &wsdom.Workspace{
				RegionalMetadata: commondomain.RegionalMetadata{
					Region: testRegion,
					CommonMetadata: commondomain.CommonMetadata{
						Name: workspaceName,
					},
					Scope: resource.Scope{
						Tenant: testTenant,
					},
				},
			}
			if err := workspaceRepo.Load(ctx, &loadedWs); err != nil {
				return false, err
			}
			if loadedWs.Status != nil && loadedWs.Status.State == commondomain.ResourceStateActive {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err, "workspace resource should become active before deletion")
		require.NotNil(t, loadedWs)
		require.NotNil(t, loadedWs.Status)
		require.NotNil(t, loadedWs.Status.State)
		require.Equal(t, commondomain.ResourceStateActive, loadedWs.Status.State)
		//
		// When we delete the workspace resource
		err = workspaceRepo.Delete(t.Context(), wsDomain)
		require.NoError(t, err)

		//
		// Then the resource should eventually be removed
		err = wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			loadedWs = &wsdom.Workspace{
				RegionalMetadata: commondomain.RegionalMetadata{
					Region: testRegion,
					CommonMetadata: commondomain.CommonMetadata{
						Name: workspaceName,
					},
					Scope: resource.Scope{
						Tenant: testTenant,
					},
				},
			}
			err := workspaceRepo.Load(ctx, &loadedWs)
			if err != nil && errors.Is(err, kernel.ErrNotFound) { // Corrected IsNotFound check
				return true, nil
			}
			if err != nil {
				return false, err
			}
			return false, nil
		})
		require.NoError(t, err, "workspace resource should be deleted")
	})
}

// TestWorkspaceRegionInSync runs a workspace past every writer the deployed stack has — the
// adapter's create and unversioned update, the delegator's finalizer and status writes, and the
// dummy plugin's own versioned update — and holds its region field and label equal after each,
// against the CRD actually deployed. The field is what the plugin provisions into, the label what routed the CR to
// this delegator, so a CR on which they disagree is reconciled here and created somewhere else.
// It then checks the deployed CRD refuses the two edits that could move the field on its own.
func TestWorkspaceRegionInSync(t *testing.T) {
	name := "test-ws-region-" + uuid.New().String()[:8]
	newWorkspace := func(labels map[string]string) *wsdom.Workspace {
		return &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				Region:         testRegion,
				CommonMetadata: commondomain.CommonMetadata{Name: name},
				Scope:          resource.Scope{Tenant: testTenant},
				Labels:         labels,
			},
		}
	}
	t.Cleanup(func() { _ = workspaceRepo.Delete(context.Background(), newWorkspace(nil)) })

	requireInSync := func(t *testing.T) {
		t.Helper()
		field, label, err := testenv.WorkspaceRegion(t.Context(), dynamicClient, testTenant, name, testRegion)
		require.NoError(t, err)
		require.Equal(t, testRegion, field, "region field")
		require.Equal(t, testRegion, label, "region label")
	}
	// waitReconciled waits for the workspace to be active with the dummy plugin's record of the
	// labels it last applied reading applied — which the plugin writes back onto the CR itself,
	// through a versioned full-replace update, only when the labels changed.
	waitReconciled := func(t *testing.T, applied string) {
		t.Helper()
		err := wait.PollUntilContextTimeout(t.Context(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
			ws := newWorkspace(nil)
			if err := workspaceRepo.Load(ctx, &ws); err != nil {
				return false, err
			}
			return ws.Status != nil && ws.Status.State == commondomain.ResourceStateActive &&
				ws.Annotations[dummyplugin.AppliedLabelsAnnotation] == applied, nil
		})
		require.NoError(t, err, "workspace should reconcile to active with the plugin having applied %q", applied)
	}

	//
	// Given a workspace created through the adapter, as the gateway creates it
	_, err := workspaceRepo.Create(t.Context(), newWorkspace(map[string]string{"step": "create"}))
	require.NoError(t, err)
	requireInSync(t)

	//
	// When the delegator has reconciled it — its finalizer and status writes, and the plugin's
	// own update recording the labels
	waitReconciled(t, "step=create")
	requireInSync(t)

	//
	// And it is updated without a resourceVersion, as a PUT with no If-Unmodified-Since is, and
	// the plugin has applied that update in turn
	_, err = workspaceRepo.Update(t.Context(), newWorkspace(map[string]string{"step": "update"}))
	require.NoError(t, err)
	requireInSync(t)
	waitReconciled(t, "step=update")
	requireInSync(t)

	//
	// Then the deployed CRD refuses moving the field, or dropping it, behind the label's back
	namespace, err := k8sadapter.ResolveNamespace(newWorkspace(nil))
	require.NoError(t, err)
	crs := dynamicClient.Resource(wsk8s.WorkspaceGVR).Namespace(namespace)
	for _, patch := range []string{`{"region":"region-two"}`, `{"region":null}`} {
		_, err := crs.Patch(t.Context(), name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
		require.Truef(t, apierrors.IsInvalid(err), "patch %s must be refused by the CRD, got %v", patch, err)
	}
	requireInSync(t)
}
