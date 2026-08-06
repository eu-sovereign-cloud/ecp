package port

import (
	"context"

	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

// RouteTableStore reconciles the SECA route table. In the POC routing is folded into the
// public LAN (Lan.Public=true), so no IONOS resource maps to a route table; this is a pure
// declaration.
type RouteTableStore interface {
	Create(ctx context.Context, domain *routetabledom.RouteTable) error
	Delete(ctx context.Context, domain *routetabledom.RouteTable) error
}
