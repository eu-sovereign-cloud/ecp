package plugin

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

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
