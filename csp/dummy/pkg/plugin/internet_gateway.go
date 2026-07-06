package plugin

import (
	"context"
	"log/slog"
	"time"

	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

type InternetGateway struct {
	logger *slog.Logger
}

func NewInternetGateway(logger *slog.Logger) *InternetGateway {
	return &InternetGateway{logger: logger}
}

func (ig *InternetGateway) Create(ctx context.Context, resource *internetgatewaydom.InternetGateway) error {
	return simulateInternetGateway(ctx, "create", resource, internetGatewayDelay(), ig.logger)
}

func (ig *InternetGateway) Delete(ctx context.Context, resource *internetgatewaydom.InternetGateway) error {
	return simulateInternetGateway(ctx, "delete", resource, internetGatewayDelay(), ig.logger)
}

func internetGatewayDelay() time.Duration {
	return networkDelay()
}
