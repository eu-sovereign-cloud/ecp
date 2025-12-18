package workspace

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type ListWorkspaces struct {
	Logger *slog.Logger
	Repo   port.ReaderRepo[*regional.WorkspaceDomain]
}

func (c ListWorkspaces) Do(ctx context.Context, params model.ListParams) ([]*regional.WorkspaceDomain, *string, error) {
	var domainWorkspaces []*regional.WorkspaceDomain
	nextSkipToken, err := c.Repo.List(ctx, params, &domainWorkspaces)
	if err != nil {
		return nil, nil, err
	}

	return domainWorkspaces, nextSkipToken, nil
}
