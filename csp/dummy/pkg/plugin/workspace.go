package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	wsdom "github.com/eu-sovereign-cloud/ecp/resources/workspace/v1"
)

type Workspace struct {
	logger *slog.Logger
}

func NewWorkspace(logger *slog.Logger) *Workspace {
	return &Workspace{logger: logger}
}

func (w *Workspace) Create(ctx context.Context, resource *wsdom.Workspace) error {
	return simulateWS(ctx, "create", resource, workspaceDelay(), w.logger)
}

func (w *Workspace) Delete(ctx context.Context, resource *wsdom.Workspace) error {
	return simulateWS(ctx, "delete", resource, workspaceDelay(), w.logger)
}

// workspaceDelay returns the simulated latency of a workspace operation.
func workspaceDelay() time.Duration {
	const base int = 15

	variation := rand.IntN(30) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	return time.Duration(base+variation) * time.Second
}
