package builder_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
)

// scopableReconciler is a controller that can be region-scoped, as every resource slice's is
// through the generic controller it embeds.
type scopableReconciler struct {
	recordingReconciler
	regions []string
}

func (r *scopableReconciler) ScopeToRegions(regions []string) { r.regions = regions }

// TestControllerSet_ScopeToRegions pins where a delegator's region scope is applied: the set
// pushes it into every controller it holds as it binds them, so a plugin's cmd/main.go adds
// controllers the same way whether or not the deployment is scoped.
func TestControllerSet_ScopeToRegions(t *testing.T) {
	t.Parallel()

	t.Run("every scopable controller is capped to the set's regions", func(t *testing.T) {
		regions := []string{"itbg-bergamo", "deff-frankfurt"}
		cs := builder.NewControllerSet()
		cs.ScopeToRegions(regions)
		first, second := &scopableReconciler{}, &scopableReconciler{}
		cs.Add(first).Add(second)

		require.NoError(t, cs.SetupWithManager(nil))
		require.Equal(t, regions, first.regions)
		require.Equal(t, regions, second.regions)
		require.True(t, first.called.Load() && second.called.Load())
	})

	t.Run("an unscoped set leaves controllers watching every region", func(t *testing.T) {
		cs := builder.NewControllerSet()
		r := &scopableReconciler{}
		cs.Add(r)

		require.NoError(t, cs.SetupWithManager(nil))
		require.Nil(t, r.regions)
	})

	t.Run("a controller that cannot be scoped is still set up", func(t *testing.T) {
		cs := builder.NewControllerSet()
		cs.ScopeToRegions([]string{"itbg-bergamo"})
		r := &recordingReconciler{}
		cs.Add(r)

		require.NoError(t, cs.SetupWithManager(nil))
		require.True(t, r.called.Load())
	})
}
