package workspace

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type UpdateWorkspace struct {
	Logger *slog.Logger
	Repo   port.WriterRepo[*regional.WorkspaceDomain]
}

func (c *UpdateWorkspace) Do(ctx context.Context, domain *regional.WorkspaceDomain) (*regional.WorkspaceDomain, error) {
	result, err := c.Repo.Update(ctx, domain)
	if err != nil {
		return nil, err
	}
	return *result, nil
}
