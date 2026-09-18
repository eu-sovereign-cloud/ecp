package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
)

// scopedResource is the smallest thing a GenericController can be instantiated for.
type scopedResource struct{}

func (scopedResource) GetName() string      { return "" }
func (scopedResource) GetVersion() string   { return "" }
func (scopedResource) GetTenant() string    { return "" }
func (scopedResource) GetWorkspace() string { return "" }

// The controller set scopes whatever implements RegionScoped, and nothing fails to compile if
// the generic controller stops doing so — every delegator would just quietly widen back to
// every region. This assertion is that missing compile error.
var _ builder.RegionScoped = (*GenericController[scopedResource])(nil)

func TestRequeueFor(t *testing.T) {
	const defaultInterval = 10 * time.Second

	logger := slog.New(slog.DiscardHandler)
	failure := errors.New("provider exploded")

	tests := []struct {
		name       string
		err        error
		wantResult ctrl.Result
		wantErr    error
	}{
		{
			name:       "a plain in-flight signal requeues at the default interval",
			err:        backend.StillProcessing,
			wantResult: ctrl.Result{RequeueAfter: defaultInterval},
		},
		{
			name:       "a signal with a duration requeues at that duration",
			err:        backend.Revisit(90 * time.Second),
			wantResult: ctrl.Result{RequeueAfter: 90 * time.Second},
		},
		{
			// Zero is "the configured cadence", not "immediately" - an immediate requeue would
			// spin the workqueue against a provider that has just told us it is not ready.
			name:       "a zero duration falls back to the default interval",
			err:        backend.Revisit(0),
			wantResult: ctrl.Result{RequeueAfter: defaultInterval},
		},
		{
			// The cause reaches the log, not the error return: a progress signal is not a failure,
			// so it must not enter the reconcile-error metric or trigger backoff.
			name:       "a signal carrying a cause still reschedules rather than failing",
			err:        backend.RevisitBecause(time.Minute, errors.New("csp unavailable")),
			wantResult: ctrl.Result{RequeueAfter: time.Minute},
		},
		{
			name:       "a refused operation stops instead of retrying",
			err:        fmt.Errorf("%w: region is immutable", backend.ErrNotSupported),
			wantResult: ctrl.Result{},
		},
		{
			// Error alone: controller-runtime discards a Result returned with a non-nil error and
			// applies exponential backoff, which is what a failing reconcile wants.
			name:       "any other failure is surfaced for backoff",
			err:        failure,
			wantResult: ctrl.Result{},
			wantErr:    failure,
		},
		{
			// The ordering rule. A plugin that wraps a refusal inside a progress signal must not
			// get an endless reschedule out of it.
			name:       "a refusal wrapped in a progress signal still stops",
			err:        backend.RevisitBecause(time.Second, fmt.Errorf("%w: flavor is immutable", backend.ErrNotSupported)),
			wantResult: ctrl.Result{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := requeueFor(context.Background(), logger, tc.err, defaultInterval)

			require.Equal(t, tc.wantResult, result)

			if tc.wantErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

// TestRegionPredicate covers the watch filter a region-scoped delegator runs with: only a CR
// whose internal region label names one of the regions it serves may reach Reconcile, on every
// event kind. A miss here either strands resources nobody reconciles or reconciles another
// deployment's region, and neither shows up until something is already in the wrong cluster.
func TestRegionPredicate(t *testing.T) {
	t.Parallel()

	labelled := func(region string) client.Object {
		obj := &metav1.PartialObjectMetadata{}
		if region != "" {
			obj.SetLabels(map[string]string{k8slabels.InternalRegionLabel: region})
		}
		return obj
	}

	t.Run("an empty scope filters nothing", func(t *testing.T) {
		pred, err := regionPredicate(nil)
		require.NoError(t, err)
		require.Nil(t, pred, "no predicate at all, so the watch stays cluster-wide")
	})

	t.Run("a region outside the scope is dropped on every event kind", func(t *testing.T) {
		pred, err := regionPredicate([]string{"itbg-bergamo", "deff-frankfurt"})
		require.NoError(t, err)

		for _, tc := range []struct {
			region string
			want   bool
		}{
			{region: "itbg-bergamo", want: true},
			{region: "deff-frankfurt", want: true},
			{region: "elsewhere", want: false},
			// A resource with no region belongs to no region, so a scoped delegator leaves
			// it alone rather than claiming it by default.
			{region: "", want: false},
		} {
			obj := labelled(tc.region)
			require.Equal(t, tc.want, pred.Create(event.CreateEvent{Object: obj}), "create %q", tc.region)
			require.Equal(t, tc.want, pred.Update(event.UpdateEvent{ObjectNew: obj}), "update %q", tc.region)
			require.Equal(t, tc.want, pred.Delete(event.DeleteEvent{Object: obj}), "delete %q", tc.region)
			require.Equal(t, tc.want, pred.Generic(event.GenericEvent{Object: obj}), "generic %q", tc.region)
		}
	})

	t.Run("a region that cannot be a label value fails setup", func(t *testing.T) {
		_, err := regionPredicate([]string{"not a label value"})
		require.Error(t, err, "a selector that can never match must not start a delegator that reconciles nothing")
	})
}
