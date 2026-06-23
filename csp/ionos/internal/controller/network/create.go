package network

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	netdom "github.com/eu-sovereign-cloud/ecp/resources/network/networks/v1"
)

type CreateNetwork struct {
	Store port.NetworkStore
}

func (c *CreateNetwork) Do(ctx context.Context, domain *netdom.Network) error {
	return c.Store.Create(ctx, domain)
}
