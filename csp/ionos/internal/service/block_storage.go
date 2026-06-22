package service

import (
	"context"

	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/domain"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/backend/kubernetes"
	blockstoragectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/block_storage"
)

var _ bsk8s.BlockStoragePlugin = (*BlockStorage)(nil)

type BlockStorage struct {
	Creator       *blockstoragectrl.CreateBlockStorage
	Deleter       *blockstoragectrl.DeleteBlockStorage
	SizeIncreaser *blockstoragectrl.IncreaseSizeBlockStorage
}

func (s *BlockStorage) Create(ctx context.Context, resource *bsdom.BlockStorageDomain) error {
	return s.Creator.Do(ctx, resource)
}

func (s *BlockStorage) Delete(ctx context.Context, resource *bsdom.BlockStorageDomain) error {
	return s.Deleter.Do(ctx, resource)
}

func (s *BlockStorage) IncreaseSize(ctx context.Context, resource *bsdom.BlockStorageDomain) error {
	return s.SizeIncreaser.Do(ctx, resource)
}
