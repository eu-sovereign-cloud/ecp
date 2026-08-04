//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	dummyplugin "github.com/eu-sovereign-cloud/ecp/csp/dummy/pkg/plugin"
)

// Updates to an already-active resource are the path these two tests cover. Creating and deleting
// are edges in the lifecycle state machine; an update is not, so before the plugin gained an
// Update operation a change to a live resource was accepted by the API, stored, echoed back on the
// next GET, and never reached the provider at all. Nothing short of the full stack catches that:
// the change has to travel API -> CR -> delegator -> plugin and back into what a GET returns,
// which is exactly what the e2e stack exercises and the isolated suites cannot.
//
// The stack here runs the dummy plugin, whose stand-in for "wrote it to the provider" is the
// applied-labels annotation it stamps back onto the resource. Aruba's real equivalent is
// spec.tags on the Aruba CR; both are driven by the same reconciler arm.

// TestUpdateLabelsReachThePlugin is the tag path: a SECA label edit has to be converted by the CSP
// plugin and land on the provider resource. On Aruba that means the VPC's tags; here it means the
// dummy plugin's applied-labels record, which the API serves back as an annotation.
func TestUpdateLabelsReachThePlugin(t *testing.T) {
	ctx := context.Background()
	networkName := "e2e-upd-labels-" + uuid.New().String()[:8]

	t.Cleanup(func() {
		_, _ = networkClient.DeleteNetworkWithResponse(ctx, testTenant, testWorkspace, networkName, nil)
	})

	network := func(labels schema.Labels) schema.Network {
		return schema.Network{
			Labels: labels,
			Spec: schema.NetworkSpec{
				Cidr:   schema.Cidr{Ipv4: "10.60.0.0/16"},
				SkuRef: schema.Reference{Resource: "sku-1"},
			},
		}
	}

	resp, err := networkClient.CreateOrUpdateNetworkWithResponse(ctx, testTenant, testWorkspace, networkName, nil,
		network(schema.Labels{"env": "staging"}))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())

	waitForActive(t, "network", func(ctx context.Context) (schema.ResourceState, bool, error) {
		r, err := networkClient.GetNetworkWithResponse(ctx, testTenant, testWorkspace, networkName)
		if err != nil {
			return "", false, err
		}
		if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
			return "", false, nil
		}
		return r.JSON200.Status.State, true, nil
	})

	// The plugin's Update runs on every reconcile of an active resource, so the labels the network
	// was created with are recorded as soon as it goes active - without anyone editing anything.
	waitForAppliedLabels(t, networkName, "env=staging")

	// Now the actual edit. Note this PUT carries no annotations, so it clears the record the
	// plugin just wrote; the plugin re-stamping it with the new labels is the convergence being
	// asserted, not an accident.
	resp, err = networkClient.CreateOrUpdateNetworkWithResponse(ctx, testTenant, testWorkspace, networkName, nil,
		network(schema.Labels{"env": "prod", "team": "platform"}))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())

	// Sorted, because the rendering has to be stable across reconciles or it would look like a
	// change on every pass and never settle.
	waitForAppliedLabels(t, networkName, "env=prod,team=platform")

	final, err := networkClient.GetNetworkWithResponse(ctx, testTenant, testWorkspace, networkName)
	require.NoError(t, err)
	require.NotNil(t, final.JSON200)

	require.Equal(t, schema.Labels{"env": "prod", "team": "platform"}, final.JSON200.Labels,
		"the edited labels must survive the round trip - including the newly added key, whose "+
			"key list lives in commonData and used to be dropped on update")

	require.Equal(t, schema.ResourceStateActive, final.JSON200.Status.State,
		"a successful update must leave the resource active")
	requireNoUpdateFailure(t, final.JSON200.Status.Conditions)
}

// TestUpdateResource is the resource path: a spec edit on a live resource. Block storage is the
// case worth pinning, because it is the one resource with two competing post-active behaviours - a
// size increase has its own operation and its own transition through "updating", and it has to win
// over the generic update arm or a pending resize would be swallowed and never applied.
func TestUpdateResource(t *testing.T) {
	ctx := context.Background()
	volumeName := "e2e-upd-resource-" + uuid.New().String()[:8]

	t.Cleanup(func() {
		_, _ = storageClient.DeleteBlockStorageWithResponse(ctx, testTenant, testWorkspace, volumeName, nil)
	})

	volume := func(sizeGB int, labels schema.Labels) schema.BlockStorage {
		return schema.BlockStorage{
			Labels: labels,
			Spec: schema.BlockStorageSpec{
				SizeGB: sizeGB,
				SkuRef: schema.Reference{Resource: "sku-1"},
			},
		}
	}

	waitActive := func() {
		waitForActive(t, "block storage", func(ctx context.Context) (schema.ResourceState, bool, error) {
			r, err := storageClient.GetBlockStorageWithResponse(ctx, testTenant, testWorkspace, volumeName)
			if err != nil {
				return "", false, err
			}
			if r.StatusCode() != http.StatusOK || r.JSON200 == nil || r.JSON200.Status == nil {
				return "", false, nil
			}
			return r.JSON200.Status.State, true, nil
		})
	}

	resp, err := storageClient.CreateOrUpdateBlockStorageWithResponse(ctx, testTenant, testWorkspace, volumeName, nil,
		volume(1, schema.Labels{"tier": "cold"}))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	waitActive()

	t.Run("a spec change is applied and the resource returns to active", func(t *testing.T) {
		resp, err := storageClient.CreateOrUpdateBlockStorageWithResponse(ctx, testTenant, testWorkspace, volumeName, nil,
			volume(2, schema.Labels{"tier": "cold"}))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		// The resize runs through "updating" and back; the generic update arm must not intercept
		// it, or the volume would sit at its old observed size forever.
		err = wait.PollUntilContextTimeout(ctx, activePollInterval, activeTimeout, true, func(ctx context.Context) (bool, error) {
			r, err := storageClient.GetBlockStorageWithResponse(ctx, testTenant, testWorkspace, volumeName)
			if err != nil {
				return false, err
			}
			if r.JSON200 == nil || r.JSON200.Status == nil {
				return false, nil
			}
			return r.JSON200.Status.State == schema.ResourceStateActive && r.JSON200.Status.SizeGB == 2, nil
		})
		require.NoError(t, err, "the resized volume should report its new size and return to active")
	})

	t.Run("a label change on the same resource reaches the plugin", func(t *testing.T) {
		resp, err := storageClient.CreateOrUpdateBlockStorageWithResponse(ctx, testTenant, testWorkspace, volumeName, nil,
			volume(2, schema.Labels{"tier": "hot", "backup": "daily"}))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		err = wait.PollUntilContextTimeout(ctx, activePollInterval, activeTimeout, true, func(ctx context.Context) (bool, error) {
			r, err := storageClient.GetBlockStorageWithResponse(ctx, testTenant, testWorkspace, volumeName)
			if err != nil {
				return false, err
			}
			if r.JSON200 == nil {
				return false, nil
			}
			return r.JSON200.Annotations[dummyplugin.AppliedLabelsAnnotation] == "backup=daily,tier=hot", nil
		})
		require.NoError(t, err, "the plugin should record the edited labels on the block storage")

		final, err := storageClient.GetBlockStorageWithResponse(ctx, testTenant, testWorkspace, volumeName)
		require.NoError(t, err)
		require.NotNil(t, final.JSON200)
		require.Equal(t, schema.ResourceStateActive, final.JSON200.Status.State)
		require.Equal(t, 2, final.JSON200.Spec.SizeGB, "the earlier resize must not be undone by a label edit")
		requireNoUpdateFailure(t, final.JSON200.Status.Conditions)
	})
}

// waitForAppliedLabels polls until the dummy plugin has recorded exactly want. The record travels
// back through the API as an annotation, so reaching it proves the whole round trip completed
// rather than merely that the gateway stored the edit.
func waitForAppliedLabels(t *testing.T, networkName, want string) {
	t.Helper()

	var last string
	err := wait.PollUntilContextTimeout(context.Background(), activePollInterval, activeTimeout, true, func(ctx context.Context) (bool, error) {
		r, err := networkClient.GetNetworkWithResponse(ctx, testTenant, testWorkspace, networkName)
		if err != nil {
			return false, err
		}
		if r.StatusCode() != http.StatusOK || r.JSON200 == nil {
			return false, nil
		}
		last = r.JSON200.Annotations[dummyplugin.AppliedLabelsAnnotation]

		return last == want, nil
	})
	require.NoErrorf(t, err, "plugin should have applied labels %q within %s, last saw %q", want, activeTimeout, last)
}

// requireNoUpdateFailure asserts the reconciler reported no failed update. A provider that cannot
// apply a change records an UpdateFailed condition and keeps the resource active, so state alone
// is not enough to tell a successful update from a refused one.
func requireNoUpdateFailure(t *testing.T, conditions []schema.StatusCondition) {
	t.Helper()

	for _, condition := range conditions {
		require.NotEqualf(t, "UpdateFailed", condition.Type,
			"the update should not have been refused: %s", condition.Message)
	}
}
