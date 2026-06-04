package network

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
)

type CreateNetwork struct {
	Store port.NetworkStore
}

func (c *CreateNetwork) Do(ctx context.Context, domain *regional.NetworkDomain) error {
	return c.Store.Create(ctx, domain)
}
