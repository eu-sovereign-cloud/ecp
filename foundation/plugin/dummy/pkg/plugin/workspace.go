package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

// Workspace is a dummy implementation of the Workspace plugin.
type Workspace struct {
	logger    *slog.Logger
	delayFunc DelayFunc
}

func NewWorkspace(logger *slog.Logger) *Workspace {
	return &Workspace{
		logger:    logger,
		delayFunc: defaultWorkspaceDelay,
	}
}

// newWorkspaceWithDelay creates a Workspace with a custom delay function.
// Intended for testing.
func newWorkspaceWithDelay(logger *slog.Logger, delay DelayFunc) *Workspace {
	return &Workspace{logger: logger, delayFunc: delay}
}

func (w *Workspace) Create(ctx context.Context, resource *regional.WorkspaceDomain) error {
	w.logger.Info("dummy workspace plugin: Create called", "resource_name", resource.GetName())
	delay := w.delayFunc()
	w.logger.Info("dummy workspace plugin: Create finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func (w *Workspace) Delete(ctx context.Context, resource *regional.WorkspaceDomain) error {
	w.logger.Info("dummy workspace plugin: Delete called", "resource_name", resource.GetName())
	delay := w.delayFunc()
	w.logger.Info("dummy workspace plugin: Delete finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func defaultWorkspaceDelay() int {
	const base = 15
	delay := base + rand.IntN(30)
	time.Sleep(time.Duration(delay) * time.Second)
	return delay
}
