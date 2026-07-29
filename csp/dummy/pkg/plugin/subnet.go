package plugin

import (
	"context"
	"log/slog"
	"time"

	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

type Subnet struct {
	logger *slog.Logger
}

func NewSubnet(logger *slog.Logger) *Subnet {
	return &Subnet{logger: logger}
}

func (s *Subnet) Create(ctx context.Context, resource *subnetdom.Subnet) error {
	return simulateSubnet(ctx, "create", resource, subnetDelay(), s.logger)
}

func (s *Subnet) Delete(ctx context.Context, resource *subnetdom.Subnet) error {
	return simulateSubnet(ctx, "delete", resource, subnetDelay(), s.logger)
}

func subnetDelay() time.Duration {
	return networkDelay()
}
