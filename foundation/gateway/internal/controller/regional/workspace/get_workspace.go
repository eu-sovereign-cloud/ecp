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

func (c *GetWorkspace) Do(ctx context.Context, ir port.IdentifiableResource) (*regional.WorkspaceDomain, error) {
	domain := &regional.WorkspaceDomain{}
	domain.Name = ir.GetName()
	domain.Tenant = ir.GetTenant()

	if err := c.Repo.Load(ctx, &domain); err != nil {
		return nil, err
	}
	return domain, nil
}
