package plugin

import (
	"context"
	"log/slog"

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
	return nil
}

func (b *BlockStorage) Delete(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("dummy block storage plugin: Delete called", "resource_name", resource.GetName())
	return nil
}

func (b *BlockStorage) IncreaseSize(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("dummy block storage plugin: IncreaseSize called", "resource_name", resource.GetName(), "new_size_gb", resource.Spec.SizeGB)
	return nil
}
