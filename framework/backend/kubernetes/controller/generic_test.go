package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
)

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
