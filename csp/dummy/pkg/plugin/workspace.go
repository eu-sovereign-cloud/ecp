package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	delegator "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/workspace/v1"
)

type Workspace struct {
	logger  *slog.Logger
	tracker *asyncTracker
}

func NewWorkspace(logger *slog.Logger) *Workspace {
	return &Workspace{logger: logger, tracker: newAsyncTracker()}
}

func (w *Workspace) Create(ctx context.Context, resource *wsdom.Workspace) error {
	return w.simulate("create", resource, workspaceDelay())
}

func (w *Workspace) Delete(ctx context.Context, resource *wsdom.Workspace) error {
	return w.simulate("delete", resource, workspaceDelay())
}

// simulate reports a long-running operation as still in progress until its
// simulated delay has elapsed, without blocking the reconciliation worker.
func (w *Workspace) simulate(op string, resource *wsdom.Workspace, delay time.Duration) error {
	key := op + ":" + resourceKey(resource)

	if !w.tracker.done(key, delay) {
		w.logger.Info("dummy workspace plugin: still processing",
			"op", op, "resource_name", resource.GetName())

		return delegator.ErrStillProcessing
	}

	w.logger.Info("dummy workspace plugin: finished",
		"op", op, "resource_name", resource.GetName())

	return nil
}

// workspaceDelay returns the simulated latency of a workspace operation.
func workspaceDelay() time.Duration {
	const base int = 15

	variation := rand.IntN(30) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	return time.Duration(base+variation) * time.Second
}
