package network

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

// CreateNetwork creates a new network resource.
type CreateNetwork struct {
	Logger      *slog.Logger
	NetworkRepo port.WriterRepo[*regional.NetworkDomain]
}

func (c CreateNetwork) Do(
	ctx context.Context, domain *regional.NetworkDomain,
) (*regional.NetworkDomain, error) {
	result, err := c.NetworkRepo.Create(ctx, domain)
	if err != nil {
		return nil, err
	}
	return *result, nil
}
