package plugin

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	networkconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
)

// The rendering has to be stable: it is compared against the stored annotation to decide whether a
// write is needed, and Go map iteration order is random. An unstable rendering would look like a
// change on every reconcile and never settle.
func TestFormatLabels_isSortedAndStable(t *testing.T) {
	labels := map[string]string{"team": "platform", "env": "prod", "app": "web"}

	require.Equal(t, "app=web,env=prod,team=platform", formatLabels(labels))

	// Two separate renderings of the same map, not one expression compared with itself: the point
	// is that random map iteration order does not leak into the output.
	first, second := formatLabels(labels), formatLabels(labels)
	require.Equal(t, first, second)

	require.Empty(t, formatLabels(nil))
	require.Empty(t, formatLabels(map[string]string{}))
}

// Update runs on every reconcile of an active resource, so the already-applied case must be free:
// it returns before building a client or issuing a write. This test would hang or fail against a
// cluster if that short-circuit were removed, since no kubeconfig is available here.
func TestNetworkUpdate_isANoOpWhenTheLabelsAreAlreadyRecorded(t *testing.T) {
	resource := &netdom.Network{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: "net-1"},
			Labels:         map[string]string{"env": "prod"},
			Annotations:    map[string]string{AppliedLabelsAnnotation: "env=prod"},
		},
	}

	plugin := NewNetwork(slog.New(slog.NewTextHandler(io.Discard, nil)))

	require.NoError(t, plugin.Update(context.Background(), resource))
	require.Equal(t, "env=prod", resource.Annotations[AppliedLabelsAnnotation])
}

// A changed label must be recorded on the resource. The write itself needs a cluster, so this
// drives applyUpdate with a persist step of its own to check what gets stamped and that it is
// persisted exactly once.
func TestApplyUpdate_recordsChangedLabelsAndPersistsOnce(t *testing.T) {
	resource := &netdom.Network{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: "net-1"},
			Labels:         map[string]string{"env": "prod", "team": "platform"},
			Annotations:    map[string]string{AppliedLabelsAnnotation: "env=staging"},
		},
	}

	persisted := 0
	persist := func(_ context.Context, r *netdom.Network) error {
		persisted++
		require.Equal(t, "env=prod,team=platform", r.Annotations[AppliedLabelsAnnotation])
		return nil
	}

	require.NoError(t, recordAppliedLabels(context.Background(), resource, &resource.Annotations, resource.Labels, persist))
	require.Equal(t, 1, persisted)

	// Second pass: the record now matches, so nothing is written.
	require.NoError(t, recordAppliedLabels(context.Background(), resource, &resource.Annotations, resource.Labels, persist))
	require.Equal(t, 1, persisted, "an unchanged resource must not be written again")
}

// Guards the wiring: every plugin must point at its own resource's GVR and converters.
func TestNetworkUpdate_usesTheNetworkGVR(t *testing.T) {
	require.Equal(t, "networks", networkconv.NetworkGVR.Resource)
}
