package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

// DelayFunc is a function that simulates a delay.
// It is injectable for testing purposes.
type DelayFunc func() int

// BlockStorage is a dummy implementation of the BlockStorage plugin.
type BlockStorage struct {
	logger    *slog.Logger
	delayFunc DelayFunc
}

func NewBlockStorage(logger *slog.Logger) *BlockStorage {
	return &BlockStorage{
		logger:    logger,
		delayFunc: defaultBlockStorageDelay,
	}
}

// newBlockStorageWithDelay creates a BlockStorage with a custom delay function.
// Intended for testing.
func newBlockStorageWithDelay(logger *slog.Logger, delay DelayFunc) *BlockStorage {
	return &BlockStorage{logger: logger, delayFunc: delay}
}

func (b *BlockStorage) Create(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("dummy block storage plugin: Create called", "resource_name", resource.GetName())
	delay := b.delayFunc()
	b.logger.Info("dummy block storage plugin: Create finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func (b *BlockStorage) Delete(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("dummy block storage plugin: Delete called", "resource_name", resource.GetName())
	delay := b.delayFunc()
	b.logger.Info("dummy block storage plugin: Delete finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func (b *BlockStorage) IncreaseSize(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("dummy block storage plugin: IncreaseSize called", "resource_name", resource.GetName(), "new_size_gb", resource.Spec.SizeGB)
	// IncreaseSize simulates a longer operation by applying the delay twice.
	delay := b.delayFunc() + b.delayFunc()
	b.logger.Info("dummy block storage plugin: IncreaseSize finished", "resource_name", resource.GetName(), "new_size_gb", resource.Spec.SizeGB, "delay(seconds)", delay)
	return nil
}

func defaultBlockStorageDelay() int {
	const base = 30
	delay := base + rand.IntN(60)
	time.Sleep(time.Duration(delay) * time.Second)
	return delay
}
