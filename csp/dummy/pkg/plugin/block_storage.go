package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	delegator "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/storage/block-storages/v1"
)

type BlockStorage struct {
	logger  *slog.Logger
	tracker *asyncTracker
}

func NewBlockStorage(logger *slog.Logger) *BlockStorage {
	return &BlockStorage{logger: logger, tracker: newAsyncTracker()}
}

func (b *BlockStorage) Create(ctx context.Context, resource *bsdom.BlockStorage) error {
	return b.simulate("create", resource, blockStorageDelay())
}

func (b *BlockStorage) Delete(ctx context.Context, resource *bsdom.BlockStorage) error {
	return b.simulate("delete", resource, blockStorageDelay())
}

func (b *BlockStorage) IncreaseSize(ctx context.Context, resource *bsdom.BlockStorage) error {
	return b.simulate("increase-size", resource, 2*blockStorageDelay())
}

// simulate reports a long-running operation as still in progress until its
// simulated delay has elapsed, without blocking the reconciliation worker.
func (b *BlockStorage) simulate(op string, resource *bsdom.BlockStorage, delay time.Duration) error {
	key := op + ":" + resourceKey(resource)

	if !b.tracker.done(key, delay) {
		b.logger.Info("dummy block storage plugin: still processing",
			"op", op, "resource_name", resource.GetName())

		return delegator.ErrStillProcessing
	}

	b.logger.Info("dummy block storage plugin: finished",
		"op", op, "resource_name", resource.GetName())

	return nil
}

// blockStorageDelay returns the simulated latency of a block storage operation.
func blockStorageDelay() time.Duration {
	const base int = 30

	variation := rand.IntN(60) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	return time.Duration(base+variation) * time.Second
}
