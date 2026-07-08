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

// blockStorageDelay returns the simulated latency of a block storage operation.
// The values are deliberately small: they only need to outlast a reconcile requeue
// (1s in the e2e delegator) so the pending→active lifecycle is exercised, while
// keeping the integration suites fast. IncreaseSize doubles this.
func blockStorageDelay() time.Duration {
	const base int = 3

	variation := rand.IntN(4) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	return time.Duration(base+variation) * time.Second
}
