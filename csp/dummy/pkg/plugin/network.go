package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

type Network struct {
	logger *slog.Logger
}

func NewNetwork(logger *slog.Logger) *Network {
	return &Network{logger: logger}
}

func (n *Network) Create(ctx context.Context, resource *netdom.Network) error {
	return simulateNet(ctx, "create", resource, networkDelay(), n.logger)
}

func (n *Network) Delete(ctx context.Context, resource *netdom.Network) error {
	return simulateNet(ctx, "delete", resource, networkDelay(), n.logger)
}

func networkDelay() time.Duration {
	const base int = 30

	variation := rand.IntN(60) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	return time.Duration(base+variation) * time.Second
}
