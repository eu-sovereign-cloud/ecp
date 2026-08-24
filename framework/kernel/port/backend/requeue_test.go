package backend_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
)

func TestRequeueSignals(t *testing.T) {
	t.Run("StillProcessing keeps the message plugin tests assert on", func(t *testing.T) {
		require.Equal(t, "operation still in progress", backend.StillProcessing.Error())
	})

	t.Run("StillProcessing requests the controller default interval", func(t *testing.T) {
		require.Zero(t, backend.StillProcessing.RequeueAfter())
	})

	// Every progress signal matches every other, so a handler can test for "is this a requeue?"
	// with the sentinel it already knows regardless of the duration the signal carries.
	t.Run("any progress signal matches StillProcessing", func(t *testing.T) {
		require.ErrorIs(t, backend.Revisit(5*time.Second), backend.StillProcessing)
		require.ErrorIs(t, backend.RevisitBecause(time.Minute, errors.New("csp down")), backend.StillProcessing)
	})

	t.Run("errors.As recovers the requested duration", func(t *testing.T) {
		var rq backend.RequeueError
		require.ErrorAs(t, backend.Revisit(5*time.Second), &rq)
		require.Equal(t, 5*time.Second, rq.RequeueAfter())
	})

	t.Run("a zero duration is the default interval, not an immediate requeue", func(t *testing.T) {
		var rq backend.RequeueError
		require.ErrorAs(t, backend.Revisit(0), &rq)
		require.Zero(t, rq.RequeueAfter())
	})

	t.Run("RevisitBecause carries its cause in the message and unwraps to it", func(t *testing.T) {
		cause := errors.New("csp unavailable")
		err := backend.RevisitBecause(30*time.Second, cause)

		require.ErrorIs(t, err, cause)
		require.Contains(t, err.Error(), "operation still in progress")
		require.Contains(t, err.Error(), "csp unavailable")
	})

	// The controller classifies failures before signals precisely because this is possible.
	// The signal must not hide the failure from errors.Is.
	t.Run("a wrapped failure stays discoverable through a progress signal", func(t *testing.T) {
		err := backend.RevisitBecause(time.Second, fmt.Errorf("%w: region is immutable", backend.ErrNotSupported))
		require.ErrorIs(t, err, backend.ErrNotSupported)
	})

	t.Run("an ordinary failure is not a progress signal", func(t *testing.T) {
		var rq backend.RequeueError
		require.NotErrorAs(t, errors.New("boom"), &rq)
		require.NotErrorIs(t, errors.New("boom"), backend.StillProcessing)
		require.NotErrorIs(t, backend.ErrNotSupported, backend.StillProcessing)
	})
}
