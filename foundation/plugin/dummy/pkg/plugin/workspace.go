package plugin

import (
	"context"
	"log/slog"

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
	return nil
}

func (w *Workspace) Delete(ctx context.Context, resource *regional.WorkspaceDomain) error {
	w.logger.Info("dummy workspace plugin: Delete called", "resource_name", resource.GetName())
	return nil
}
