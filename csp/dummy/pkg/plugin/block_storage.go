package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

type BlockStorage struct {
	logger *slog.Logger
}

func NewBlockStorage(logger *slog.Logger) *BlockStorage {
	return &BlockStorage{logger: logger}
}

func (b *BlockStorage) Create(ctx context.Context, resource *bsdom.BlockStorage) error {
	return simulateBS(ctx, "create", resource, blockStorageDelay(), b.logger)
}

func (b *BlockStorage) Delete(ctx context.Context, resource *bsdom.BlockStorage) error {
	return simulateBS(ctx, "delete", resource, blockStorageDelay(), b.logger)
}

func (b *BlockStorage) IncreaseSize(ctx context.Context, resource *bsdom.BlockStorage) error {
	return simulateBS(ctx, "increase-size", resource, 2*blockStorageDelay(), b.logger)
}

func blockStorageDelay() time.Duration {
	const base int = 30

	variation := rand.IntN(60) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	return time.Duration(base+variation) * time.Second
}
