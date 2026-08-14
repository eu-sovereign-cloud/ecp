package service

import (
	"context"

	routetablectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/route_table"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	routetablek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
)

var _ routetablek8s.RouteTablePlugin = (*RouteTable)(nil)

type RouteTable struct {
	Creator *routetablectrl.CreateRouteTable
	Deleter *routetablectrl.DeleteRouteTable
}

func (r *RouteTable) Update(ctx context.Context, resource *routetabledom.RouteTable) error {
	// TODO implement me
	panic("implement me")
}

func (r *RouteTable) Create(ctx context.Context, resource *routetabledom.RouteTable) error {
	return r.Creator.Do(ctx, resource)
}

func (r *RouteTable) Delete(ctx context.Context, resource *routetabledom.RouteTable) error {
	return r.Deleter.Do(ctx, resource)
}
