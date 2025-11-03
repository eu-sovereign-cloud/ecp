package regionalprovider

import (
	"context"
	"fmt"
	"log/slog"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"
	"k8s.io/client-go/rest"

	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/delegator/api/block-storage/skus/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/provider/common"
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

// StorageController implements the StorageProvider interface and provides methods to interact with the Storage CRDs and XRDs in the Kubernetes cluster.
type StorageController struct {
	client *kubeclient.KubeClient
	logger *slog.Logger
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

// NewStorageController creates a new StorageController with a Kubernetes client.
func NewStorageController(logger *slog.Logger, cfg *rest.Config) (*StorageController, error) {
	client, err := kubeclient.NewFromConfig(cfg)
	if err != nil {
		logger.Error("failed to create kubeclient", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create kubeclient: %w", err)
	}

	return &StorageController{
		client: client,
		logger: logger.With(slog.String("component", "StorageController")),
	}, nil
}

const tenantLabelKey = "secapi.cloud/tenant-id"

func (c StorageController) ListSKUs(ctx context.Context, tenantID string, params sdkstorage.ListSkusParams) (
	*sdkstorage.SkuIterator, error,
) {
	limit := validation.GetLimit(params.Limit)

	convert := common.Adapter(func(crdStorageSKU skuv1.StorageSKU) (sdkschema.StorageSku, error) {
		return fromCRToSDKStorageSKU(crdStorageSKU), nil
	})
	opts := common.NewListOptions().Namespace(tenantID)
	if limit > 0 {
		opts.Limit(limit)
	}
	if params.SkipToken != nil {
		opts.SkipToken(*params.SkipToken)
	}
	if params.Labels != nil {
		opts.Selector(*params.Labels)
	}

	sdkStorageSKUs, nextSkipToken, err := common.ListResources(ctx, c.client.Client, skuv1.StorageSKUGVR, *c.logger, convert, opts)
	if err != nil {
		return nil, err
	}

	iterator := sdkstorage.SkuIterator{
		Items: sdkStorageSKUs,
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
	convert := common.Adapter(func(crdStorageSKU skuv1.StorageSKU) (sdkschema.StorageSku, error) {
		return fromCRToSDKStorageSKU(crdStorageSKU), nil
	})
	opts := common.NewGetOptions().Namespace(tenantID)
	sku, err := common.GetResource(ctx, c.client.Client, skuv1.StorageSKUGVR, skuID, *c.logger, convert, opts)
	if err != nil {
		return nil, err
	}
	return &sku, nil
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
