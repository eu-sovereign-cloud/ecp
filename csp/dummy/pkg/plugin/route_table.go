package plugin

import (
	"context"
	"log/slog"
	"time"

	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

type RouteTable struct {
	logger *slog.Logger
}

func NewRouteTable(logger *slog.Logger) *RouteTable {
	return &RouteTable{logger: logger}
}

func (rt *RouteTable) Create(ctx context.Context, resource *routetabledom.RouteTable) error {
	return simulateRouteTable(ctx, "create", resource, routeTableDelay(), rt.logger)
}

func (rt *RouteTable) Delete(ctx context.Context, resource *routetabledom.RouteTable) error {
	return simulateRouteTable(ctx, "delete", resource, routeTableDelay(), rt.logger)
}

func routeTableDelay() time.Duration {
	return networkDelay()
}
