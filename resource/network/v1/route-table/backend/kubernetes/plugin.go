package kubernetes

import (
	"context"

	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

// RouteTablePlugin is implemented by CSP plugins that manage RouteTable resources.
type RouteTablePlugin interface {
	Create(ctx context.Context, resource *routetabledom.RouteTable) error
	Delete(ctx context.Context, resource *routetabledom.RouteTable) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *routetabledom.RouteTable) error
}
