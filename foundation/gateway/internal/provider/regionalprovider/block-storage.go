package regionalprovider

import (
	"context"
	"log/slog"

	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/block-storage/skus/v1"
	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

const (
	StorageBaseURL      = "/providers/seca.storage"
	ProviderStorageName = "seca.storage/v1"
)

type StorageSKUProvider interface {
	ListSKUs(ctx context.Context, tenantID string, params sdkstorage.ListSkusParams) (*sdkstorage.SkuIterator, error)
	GetSKU(ctx context.Context, tenantID, skuID string) (*sdkschema.StorageSku, error)
}

type ImageProvider interface {
	CreateOrUpdateImage(ctx context.Context, tenantID, imageID string, params sdkstorage.CreateOrUpdateImageParams, req sdkschema.Image,
	) (*sdkschema.Image, bool, error)
	GetImage(ctx context.Context, tenantID, imageID string) (*sdkschema.Image, error)
	DeleteImage(ctx context.Context, tenantID, imageID string, params sdkstorage.DeleteImageParams) error
	ListImages(
		ctx context.Context, tenantID string, params sdkstorage.ListImagesParams,
	) (*secapi.Iterator[sdkschema.Image], error)
}

type StorageProvider interface {
	StorageSKUProvider
	ImageProvider

	ListBlockStorages(ctx context.Context, tenantID, workspaceID string, params sdkstorage.ListBlockStoragesParams,
	) (*secapi.Iterator[sdkschema.BlockStorage], error)
	GetBlockStorage(ctx context.Context, tenantID, workspaceID, storageID string) (*sdkschema.BlockStorage, error)
	CreateOrUpdateBlockStorage(ctx context.Context, tenantID, workspaceID, storageID string,
		params sdkstorage.CreateOrUpdateBlockStorageParams, req sdkschema.BlockStorage) (*sdkschema.BlockStorage, bool, error)
	DeleteBlockStorage(ctx context.Context, tenantID, workspaceID,
		storageID string, params sdkstorage.DeleteBlockStorageParams) error
}

var _ StorageProvider = (*StorageController)(nil) // Ensure StorageController implements the StorageProvider interface.

// StorageController implements the StorageProvider interface
type StorageController struct {
	logger         *slog.Logger
	storageSKURepo port.ResourceQueryRepository[*regional.StorageSKUDomain]
}

func (c StorageController) CreateOrUpdateImage(
	ctx context.Context, tenantID, imageID string, params sdkstorage.CreateOrUpdateImageParams, req sdkschema.Image,
) (*sdkschema.Image, bool, error) {
	// TODO implement me
	panic("implement me")
}

func (c StorageController) GetImage(ctx context.Context, tenantID, imageID string) (*sdkschema.Image, error) {
	// TODO implement me
	panic("implement me")
}

func (c StorageController) DeleteImage(
	ctx context.Context, tenantID, imageID string, params sdkstorage.DeleteImageParams,
) error {
	// TODO implement me
	panic("implement me")
}

func (c StorageController) ListImages(
	ctx context.Context, tenantID string, params sdkstorage.ListImagesParams,
) (*secapi.Iterator[sdkschema.Image], error) {
	// TODO implement me
	panic("implement me")
}

// NewStorageController creates a new StorageController.
func NewStorageController(
	logger *slog.Logger,
	storageSKURepo port.ResourceQueryRepository[*regional.StorageSKUDomain],
) *StorageController {
	return &StorageController{
		logger:         logger.With(slog.String("component", "StorageController")),
		storageSKURepo: storageSKURepo,
	}
}

const tenantLabelKey = "secapi.cloud/tenant-id"

func (c StorageController) ListSKUs(ctx context.Context, tenantID string, params sdkstorage.ListSkusParams) (
	*sdkstorage.SkuIterator, error,
) {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	listParams := model.ListParams{
		Namespace: tenantID,
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}
	var domainSKUs []*regional.StorageSKUDomain
	nextSkipToken, err := c.storageSKURepo.List(ctx, listParams, &domainSKUs)
	if err != nil {
		return nil, err
	}

	// convert to sdk slice
	sdkSKUs := make([]sdkschema.StorageSku, len(domainSKUs))
	for i := range domainSKUs {
		mapped := regional.ToSDKStorageSKU(domainSKUs[i])
		sdkSKUs[i] = *mapped
	}

	iterator := sdkstorage.SkuIterator{
		Items: sdkSKUs,
		Metadata: sdkschema.ResponseMetadata{
			Provider: ProviderStorageName,
			Resource: skuv1.StorageSKUResource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return &iterator, nil
}

func (c StorageController) GetSKU(
	ctx context.Context, tenantID, skuID string,
) (*sdkschema.StorageSku, error) {
	domain := &regional.StorageSKUDomain{}
	domain.SetName(skuID)
	domain.SetNamespace(tenantID) // ensure namespaced SKU retrieval
	if err := c.storageSKURepo.Load(ctx, &domain); err != nil {
		return nil, err
	}
	return regional.ToSDKStorageSKU(domain), nil
}

func (c StorageController) ListBlockStorages(
	ctx context.Context, tenantID, workspaceID string, params sdkstorage.ListBlockStoragesParams,
) (*secapi.Iterator[sdkschema.BlockStorage], error) {
	// TODO implement me
	panic("implement me")
}

func (c StorageController) GetBlockStorage(
	ctx context.Context, tenantID, workspaceID, storageID string,
) (*sdkschema.BlockStorage, error) {
	// TODO implement me
	panic("implement me")
}

func (c StorageController) CreateOrUpdateBlockStorage(
	ctx context.Context, tenantID, workspaceID, storageID string,
	params sdkstorage.CreateOrUpdateBlockStorageParams, req sdkschema.BlockStorage,
) (*sdkschema.BlockStorage, bool, error) {
	// TODO implement me
	panic("implement me")
}

func (c StorageController) DeleteBlockStorage(
	ctx context.Context, tenantID, workspaceID, storageID string, params sdkstorage.DeleteBlockStorageParams,
) error {
	// TODO implement me
	panic("implement me")
}

func fromCRToSDKStorageSKU(crStorageSKU skuv1.StorageSKU) sdkschema.StorageSku {
	sdkStorageSKU := sdkschema.StorageSku{
		Spec: &sdkschema.StorageSkuSpec{
			Iops:          crStorageSKU.Spec.Iops,
			MinVolumeSize: crStorageSKU.Spec.MinVolumeSize,
			Type:          sdkschema.StorageSkuSpecType(crStorageSKU.Spec.Type),
		},
		Metadata: &sdkschema.SkuResourceMetadata{
			Name: crStorageSKU.GetName(),
		},
	}
	return sdkStorageSKU
}
