package workspace

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type ListWorkspace struct {
	Logger *slog.Logger
	Repo   port.ReaderRepo[*regional.WorkspaceDomain]
}

func (c *ListWorkspace) Do(ctx context.Context, params model.ListParams) ([]*regional.WorkspaceDomain, *string, error) {
	var domains []*regional.WorkspaceDomain

	skipToken, err := c.Repo.List(ctx, params, &domains)
	if err != nil {
		return nil, nil, err
	}
	return domains, skipToken, nil
}
