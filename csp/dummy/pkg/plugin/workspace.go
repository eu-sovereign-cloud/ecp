package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
)

type Workspace struct {
	logger *slog.Logger
}

func NewWorkspace(logger *slog.Logger) *Workspace {
	return &Workspace{logger: logger}
}

func (w *Workspace) Create(ctx context.Context, resource *regional.WorkspaceDomain) error {
	w.logger.Info("dummy workspace plugin: Create called", "resource_name", resource.GetName())
	delay, err := workspaceDelay(ctx)
	if err != nil {
		return err
	}
	w.logger.Info("dummy workspace plugin: Create finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func (w *Workspace) Delete(ctx context.Context, resource *regional.WorkspaceDomain) error {
	w.logger.Info("dummy workspace plugin: Delete called", "resource_name", resource.GetName())
	delay, err := workspaceDelay(ctx)
	if err != nil {
		return err
	}
	w.logger.Info("dummy workspace plugin: Delete finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func workspaceDelay(ctx context.Context) (int, error) {
	const base int = 15

	variation := rand.IntN(30) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	delay := base + variation
	select {
	case <-time.After(time.Duration(delay) * time.Second):
		return delay, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}
