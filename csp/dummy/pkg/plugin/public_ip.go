package plugin

import (
	"context"
	"log/slog"
	"time"

	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

type PublicIp struct {
	logger *slog.Logger
}

func NewPublicIp(logger *slog.Logger) *PublicIp {
	return &PublicIp{logger: logger}
}

func (p *PublicIp) Create(ctx context.Context, resource *publicipdom.PublicIp) error {
	return simulatePublicIp(ctx, "create", resource, publicIpDelay(), p.logger)
}

func (p *PublicIp) Delete(ctx context.Context, resource *publicipdom.PublicIp) error {
	return simulatePublicIp(ctx, "delete", resource, publicIpDelay(), p.logger)
}

func publicIpDelay() time.Duration {
	return networkDelay()
}
