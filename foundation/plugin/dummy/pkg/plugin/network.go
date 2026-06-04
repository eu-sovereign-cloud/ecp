package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

type Network struct {
	logger *slog.Logger
}

func NewNetwork(logger *slog.Logger) *Network {
	return &Network{logger: logger}
}

func (n *Network) Create(ctx context.Context, resource *regional.NetworkDomain) error {
	n.logger.Info("dummy network plugin: Create called", "resource_name", resource.GetName())
	delay := networkDelay()
	n.logger.Info("dummy network plugin: Create finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func (n *Network) Delete(ctx context.Context, resource *regional.NetworkDomain) error {
	n.logger.Info("dummy network plugin: Delete called", "resource_name", resource.GetName())
	delay := networkDelay()
	n.logger.Info("dummy network plugin: Delete finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func networkDelay() int {
	const base int = 30

	variation := rand.IntN(60) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	delay := base + variation
	time.Sleep(time.Duration(delay) * time.Second)

	return delay
}
