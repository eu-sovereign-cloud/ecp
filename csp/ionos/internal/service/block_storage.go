package service

import (
	"context"

	blockstoragectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/block_storage"
	delegatorplugin "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
)

var _ delegatorplugin.BlockStorage = (*BlockStorage)(nil)

type BlockStorage struct {
	Creator       *blockstoragectrl.CreateBlockStorage
	Deleter       *blockstoragectrl.DeleteBlockStorage
	SizeIncreaser *blockstoragectrl.IncreaseSizeBlockStorage
}

func (s *BlockStorage) Create(ctx context.Context, resource *regional.BlockStorageDomain) error {
	return s.Creator.Do(ctx, resource)
}

func (s *BlockStorage) Delete(ctx context.Context, resource *regional.BlockStorageDomain) error {
	return s.Deleter.Do(ctx, resource)
}

func (s *BlockStorage) IncreaseSize(ctx context.Context, resource *regional.BlockStorageDomain) error {
	return s.SizeIncreaser.Do(ctx, resource)
}
