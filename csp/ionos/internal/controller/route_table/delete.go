package routetable

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

type DeleteRouteTable struct {
	Store port.RouteTableStore
}

func (c *DeleteRouteTable) Do(ctx context.Context, domain *routetabledom.RouteTable) error {
	return c.Store.Delete(ctx, domain)
}
