package backend

import (
	"errors"
	"time"
)

// stillProcessingMessage is the text every progress signal leads with. Plugin test suites assert
// on it (csp/aruba/pkg/adapter/handler/*_test.go), so it is part of the contract.
const stillProcessingMessage = "operation still in progress"

// RequeueError is a progress signal, not a failure. A plugin returns one when the pass advanced
// the resource but did not settle it: the controller reschedules instead of recording an error,
// and the reconcile is not counted as failed.
//
// It is named without an Err prefix on purpose, following io.EOF and fs.SkipDir.
type RequeueError interface {
	error

	// RequeueAfter reports when to come back. Zero means the controller's configured default
	// interval; it never means "immediately", which would hot-loop the workqueue.
	RequeueAfter() time.Duration
}

// StillProcessing requeues at the controller's default interval with no recorded cause. It is the
// plain "in flight" signal and the one a plugin reaches for when it has no cadence opinion.
var StillProcessing RequeueError = &requeueError{}

// Revisit returns a progress signal asking the controller to come back after d. A zero d uses the
// controller's configured default interval.
func Revisit(d time.Duration) error {
	return &requeueError{after: d}
}

// RevisitBecause returns a progress signal asking the controller to come back after d, carrying
// cause for the log.
//
// cause is reported, not treated as a failure. Choosing a progress signal is an explicit opt-out
// of the workqueue's exponential backoff in favour of a cadence the plugin controls: return the
// plain error instead when backoff is what you want.
func RevisitBecause(d time.Duration, cause error) error {
	return &requeueError{after: d, cause: cause}
}

// requeueError is the only RequeueError implementation. It mirrors controller-runtime's
// terminalError: a wrapping error with a custom Is so the sentinel matches the whole family.
type requeueError struct {
	after time.Duration
	cause error
}

func (e *requeueError) RequeueAfter() time.Duration { return e.after }

func (e *requeueError) Error() string {
	if e.cause == nil {
		return stillProcessingMessage
	}

	return stillProcessingMessage + ": " + e.cause.Error()
}

// Unwrap exposes the cause so errors.Is and errors.As reach it. This is what makes the ordering
// rule in the controller load-bearing: a failure wrapped in a signal is still discoverable, so
// failures must be classified first.
func (e *requeueError) Unwrap() error { return e.cause }

// Is matches any RequeueError, so errors.Is(err, StillProcessing) holds for every progress signal
// whatever duration or cause it carries.
func (e *requeueError) Is(target error) bool {
	var t RequeueError

	return errors.As(target, &t)
}
