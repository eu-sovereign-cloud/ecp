package workspace

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

type CreateWorkspace struct {
	Logger *slog.Logger
	Repo   port.WriterRepo[*regional.WorkspaceDomain]
}

func (c *CreateWorkspace) Do(ctx context.Context, domain *regional.WorkspaceDomain) (*regional.WorkspaceDomain, error) {
	result, err := c.Repo.Create(ctx, domain)
	if err != nil {
		return nil, err
	}

	return *result, nil
}
