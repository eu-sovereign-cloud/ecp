package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
)

type Image struct {
	logger *slog.Logger
}

func NewImage(logger *slog.Logger) *Image {
	return &Image{logger: logger}
}

func (i *Image) Create(ctx context.Context, resource *imgdom.Image) error {
	return simulateImage(ctx, "create", resource, imageDelay(), i.logger)
}

func (i *Image) Delete(ctx context.Context, resource *imgdom.Image) error {
	return simulateImage(ctx, "delete", resource, imageDelay(), i.logger)
}

// imageDelay returns the simulated latency of an image operation.
func imageDelay() time.Duration {
	const base int = 30

	variation := rand.IntN(60) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	return time.Duration(base+variation) * time.Second
}
