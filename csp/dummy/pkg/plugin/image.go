package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
)

type Image struct {
	logger *slog.Logger
}

func NewImage(logger *slog.Logger) *Image {
	return &Image{logger: logger}
}

func (i *Image) Create(ctx context.Context, resource *regional.ImageDomain) error {
	i.logger.Info("dummy image plugin: Create called", "resource_name", resource.GetName())
	delay := imageDelay()
	i.logger.Info("dummy image plugin: Create finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func (i *Image) Delete(ctx context.Context, resource *regional.ImageDomain) error {
	i.logger.Info("dummy image plugin: Delete called", "resource_name", resource.GetName())
	delay := imageDelay()
	i.logger.Info("dummy image plugin: Delete finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func imageDelay() int {
	const base int = 30

	variation := rand.IntN(60) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	delay := base + variation

	time.Sleep(time.Duration(delay) * time.Second)

	return delay
}
