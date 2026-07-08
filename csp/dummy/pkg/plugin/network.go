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

// networkDelay returns the (short) simulated latency of a network operation; nic,
// internet gateway and public IP reuse it. See blockStorageDelay for why the values
// are kept small.
func networkDelay() time.Duration {
	const base int = 3

	variation := rand.IntN(4) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	return time.Duration(base+variation) * time.Second
}
