package workspace

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type DeleteWorkspace struct {
	Logger *slog.Logger
	Repo   port.WriterRepo[*regional.WorkspaceDomain]
}

func (c *DeleteWorkspace) Do(ctx context.Context) error {
	return nil
}
