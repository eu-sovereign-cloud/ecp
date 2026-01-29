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

	// Instead of deleting the Workspace CR immediately, mark it as Deleting so
	// the delegator controller can run plugin.Delete (which will remove finalizers
	// and handle external cleanup). This avoids removing the CR before the plugin
	// has a chance to perform cleanup.
	state := regional.ResourceStateDeleting
	domain.Status.State = &state

	_, err := c.Repo.Update(ctx, domain)
	return err
}
