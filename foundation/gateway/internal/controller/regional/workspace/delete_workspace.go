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

func (c *DeleteWorkspace) Do(ctx context.Context, ir port.IdentifiableResource) error {
	domain := &regional.WorkspaceDomain{}
	domain.Name = ir.GetName()
	domain.Tenant = ir.GetTenant()
	domain.ResourceVersion = ir.GetVersion()
	domain.Workspace = ir.GetWorkspace()

	return c.Repo.Delete(ctx, domain)
}
