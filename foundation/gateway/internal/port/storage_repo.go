package port

import (
	"context"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"
)

type ImageProvider interface {
	CreateOrUpdateImage(ctx context.Context, tenantID, imageID string, params storage.CreateOrUpdateImageParams, req schema.Image,
	) (*schema.Image, bool, error)
	GetImage(ctx context.Context, tenantID, imageID string) (*schema.Image, error)
	DeleteImage(ctx context.Context, tenantID, imageID string, params storage.DeleteImageParams) error
	ListImages(
		ctx context.Context, tenantID string, params storage.ListImagesParams,
	) (*secapi.Iterator[schema.Image], error)
}

type StorageProvider interface {
	StorageSKUProvider
	ImageProvider

	ListBlockStorages(ctx context.Context, tenantID, workspaceID string, params storage.ListBlockStoragesParams,
	) (*secapi.Iterator[schema.BlockStorage], error)
	GetBlockStorage(ctx context.Context, tenantID, workspaceID, storageID string) (*schema.BlockStorage, error)
	CreateOrUpdateBlockStorage(ctx context.Context, tenantID, workspaceID, storageID string,
		params storage.CreateOrUpdateBlockStorageParams, req schema.BlockStorage) (*schema.BlockStorage, bool, error)
	DeleteBlockStorage(ctx context.Context, tenantID, workspaceID,
		storageID string, params storage.DeleteBlockStorageParams) error
}

type StorageSKUProvider interface {
	ListSKUs(ctx context.Context, tenantID string, params storage.ListSkusParams) (*storage.SkuIterator, error)
	GetSKU(ctx context.Context, tenantID, skuID string) (*schema.StorageSku, error)
}
