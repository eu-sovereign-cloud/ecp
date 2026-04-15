package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

type Workspace struct {
	logger *slog.Logger
}

func NewWorkspace(logger *slog.Logger) *Workspace {
	return &Workspace{logger: logger}
}

func (w *Workspace) Create(ctx context.Context, resource *regional.WorkspaceDomain) error {
	w.logger.Info("dummy workspace plugin: Create called", "resource_name", resource.GetName())
	delay := workspaceDelay()
	w.logger.Info("dummy workspace plugin: Create finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func (w *Workspace) Delete(ctx context.Context, resource *regional.WorkspaceDomain) error {
	w.logger.Info("dummy workspace plugin: Delete called", "resource_name", resource.GetName())
	delay := workspaceDelay()
	w.logger.Info("dummy workspace plugin: Delete finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func workspaceDelay() int {
	const base int = 15

	variation := rand.IntN(30) //nolint:gosec // math/rand/v2 is fine here: delay jitter is not security-sensitive

	delay := base + variation
	time.Sleep(time.Duration(delay) * time.Second)

	return delay
}
