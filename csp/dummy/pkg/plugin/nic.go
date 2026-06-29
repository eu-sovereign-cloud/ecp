package plugin

import (
	"context"
	"log/slog"
	"time"

	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

type Nic struct {
	logger *slog.Logger
}

func NewNic(logger *slog.Logger) *Nic {
	return &Nic{logger: logger}
}

func (n *Nic) Create(ctx context.Context, resource *nicdom.Nic) error {
	return simulateNic(ctx, "create", resource, nicDelay(), n.logger)
}

func (n *Nic) Delete(ctx context.Context, resource *nicdom.Nic) error {
	return simulateNic(ctx, "delete", resource, nicDelay(), n.logger)
}

func nicDelay() time.Duration {
	return networkDelay()
}
