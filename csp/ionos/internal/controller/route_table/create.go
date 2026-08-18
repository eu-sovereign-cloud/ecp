package routetable

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

type CreateRouteTable struct {
	Store port.RouteTableStore
}

func (c *CreateRouteTable) Do(ctx context.Context, domain *routetabledom.RouteTable) error {
	return c.Store.Create(ctx, domain)
}
