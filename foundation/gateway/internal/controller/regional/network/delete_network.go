package network

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/persistence/port"
)

// DeleteNetwork deletes a network resource by identity.
type DeleteNetwork struct {
	Logger      *slog.Logger
	NetworkRepo port.WriterRepo[*regional.NetworkDomain]
}

func (c DeleteNetwork) Do(ctx context.Context, ir port.IdentifiableResource) error {
	domain := &regional.NetworkDomain{}
	domain.Name = ir.GetName()
	domain.Tenant = ir.GetTenant()
	domain.Workspace = ir.GetWorkspace()
	domain.ResourceVersion = ir.GetVersion()

	return c.NetworkRepo.Delete(ctx, domain)
}
