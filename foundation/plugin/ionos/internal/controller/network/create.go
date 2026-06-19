package network

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/pkg/port"
)

type CreateNetwork struct {
	Store port.NetworkStore
}

func (c *CreateNetwork) Do(ctx context.Context, domain *regional.NetworkDomain) error {
	return c.Store.Create(ctx, domain)
}
