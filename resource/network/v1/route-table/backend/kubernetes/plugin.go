package kubernetes

import (
	"context"

	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

// RouteTablePlugin is implemented by CSP plugins that manage RouteTable resources.
type RouteTablePlugin interface {
	Create(ctx context.Context, resource *routetabledom.RouteTable) error
	Delete(ctx context.Context, resource *routetabledom.RouteTable) error
}
