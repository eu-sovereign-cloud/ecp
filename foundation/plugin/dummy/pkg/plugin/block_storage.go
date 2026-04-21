package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

type BlockStorage struct {
	logger *slog.Logger
}

func NewBlockStorage(logger *slog.Logger) *BlockStorage {
	return &BlockStorage{logger: logger}
}

func (b *BlockStorage) Create(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("dummy block storage plugin: Create called", "resource_name", resource.GetName())
	delay := blockStorageDelay()
	b.logger.Info("dummy block storage plugin: Create finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func (b *BlockStorage) Delete(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("dummy block storage plugin: Delete called", "resource_name", resource.GetName())
	delay := blockStorageDelay()
	b.logger.Info("dummy block storage plugin: Delete finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func (b *BlockStorage) IncreaseSize(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("dummy block storage plugin: IncreaseSize called", "resource_name", resource.GetName(), "new_size_gb", resource.Spec.SizeGB)
	delay := blockStorageDelay()
	delay += blockStorageDelay()
	b.logger.Info("dummy block storage plugin: IncreaseSize finished", "resource_name", resource.GetName(), "new_size_gb", resource.Spec.SizeGB, "delay(seconds)", delay)
	return nil
}

func blockStorageDelay() int {
	const base int = 30

	variation := rand.IntN(60) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	delay := base + variation

	time.Sleep(time.Duration(delay) * time.Second)

	return delay
}
