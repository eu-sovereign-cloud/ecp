package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"
)

type BlockStorage struct {
	logger *slog.Logger
}

func NewBlockStorage(logger *slog.Logger) *BlockStorage {
	return &BlockStorage{logger: logger}
}

func (b *BlockStorage) Create(ctx context.Context, resource *bsdom.BlockStorageDomain) error {
	b.logger.Info("dummy block storage plugin: Create called", "resource_name", resource.GetName())
	delay, err := blockStorageDelay(ctx)
	if err != nil {
		return err
	}
	b.logger.Info("dummy block storage plugin: Create finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func (b *BlockStorage) Delete(ctx context.Context, resource *bsdom.BlockStorageDomain) error {
	b.logger.Info("dummy block storage plugin: Delete called", "resource_name", resource.GetName())
	delay, err := blockStorageDelay(ctx)
	if err != nil {
		return err
	}
	b.logger.Info("dummy block storage plugin: Delete finished", "resource_name", resource.GetName(), "delay(seconds)", delay)
	return nil
}

func (b *BlockStorage) IncreaseSize(ctx context.Context, resource *bsdom.BlockStorageDomain) error {
	b.logger.Info("dummy block storage plugin: IncreaseSize called", "resource_name", resource.GetName(), "new_size_gb", resource.Spec.SizeGB)
	d1, err := blockStorageDelay(ctx)
	if err != nil {
		return err
	}
	d2, err := blockStorageDelay(ctx)
	if err != nil {
		return err
	}
	b.logger.Info("dummy block storage plugin: IncreaseSize finished", "resource_name", resource.GetName(), "new_size_gb", resource.Spec.SizeGB, "delay(seconds)", d1+d2)
	return nil
}

func blockStorageDelay(ctx context.Context) (int, error) {
	const base int = 30

	variation := rand.IntN(60) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	delay := base + variation
	select {
	case <-time.After(time.Duration(delay) * time.Second):
		return delay, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}
