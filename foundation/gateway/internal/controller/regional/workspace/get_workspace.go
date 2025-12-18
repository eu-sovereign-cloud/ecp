package workspace

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type GetWorkspace struct {
	Logger *slog.Logger
	Repo   port.ReaderRepo[*regional.WorkspaceDomain]
}

func (c GetWorkspace) Do(ctx context.Context, ir port.IdentifiableResource) (*regional.WorkspaceDomain, error) {
	domainWorkspace := &regional.WorkspaceDomain{}
	domainWorkspace.SetTenant(ir.GetTenant())
	domainWorkspace.SetName(ir.GetName())

	if err := c.Repo.Load(ctx, &domainWorkspace); err != nil {
		return nil, err
	}
	return domainWorkspace, nil
}
