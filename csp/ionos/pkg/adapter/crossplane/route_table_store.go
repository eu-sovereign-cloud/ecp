package crossplane

import (
	"context"
	"log/slog"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

var _ port.RouteTableStore = (*RouteTableStore)(nil)

type RouteTableStore struct {
	base
}

func NewRouteTableStore(c client.Client, logger *slog.Logger) *RouteTableStore {
	return &RouteTableStore{base{client: c, logger: logger}}
}

// Create marks the route table ready. For the POC routing is folded into the IONOS public LAN
// (created by the Network plugin), so no IONOS resource maps to a SECA route table; it only
// needs to exist as a ready declaration.
func (a *RouteTableStore) Create(ctx context.Context, domain *routetabledom.RouteTable) error {
	a.logger.Info("route table: ready as declaration, no IONOS resource (folded into the public LAN)",
		"route_table", domain.GetName())
	return nil
}

// Delete is a no-op: there is no backing IONOS resource to remove.
func (a *RouteTableStore) Delete(ctx context.Context, domain *routetabledom.RouteTable) error {
	return nil
}
