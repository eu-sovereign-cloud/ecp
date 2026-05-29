package network

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

// GetNetwork retrieves a network resource by identity.
type GetNetwork struct {
	Logger      *slog.Logger
	NetworkRepo port.ReaderRepo[*regional.NetworkDomain]
}

func (c GetNetwork) Do(
	ctx context.Context, ir port.IdentifiableResource,
) (*regional.NetworkDomain, error) {
	domain := &regional.NetworkDomain{}
	domain.Name = ir.GetName()
	domain.Tenant = ir.GetTenant()
	domain.Workspace = ir.GetWorkspace()
	domain.ResourceVersion = ir.GetVersion()

	if err := c.NetworkRepo.Load(ctx, &domain); err != nil {
		return nil, err
	}
	return domain, nil
}
