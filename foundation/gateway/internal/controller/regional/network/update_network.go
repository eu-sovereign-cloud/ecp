package network

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

// UpdateNetwork updates an existing network resource.
type UpdateNetwork struct {
	Logger      *slog.Logger
	NetworkRepo port.WriterRepo[*regional.NetworkDomain]
}

func (c UpdateNetwork) Do(
	ctx context.Context, domain *regional.NetworkDomain,
) (*regional.NetworkDomain, error) {
	result, err := c.NetworkRepo.Update(ctx, domain)
	if err != nil {
		return nil, err
	}
	return *result, nil
}
