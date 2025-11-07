package regionalprovider

import (
	"context"
	"fmt"
	"log/slog"

	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/block-storage/skus/v1"
	storagev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/block-storage/storages/v1"
	regionalcommon "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/common"

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
	// Import the Storage CRD type

	// First, try to get the existing resource
	existingStorage, err := c.getBlockStorageCR(ctx, tenantID, storageID)
	isUpdate := err == nil && existingStorage != nil

	// Create the Storage CR from the SDK request
	storageCR := fromSDKToStorageCR(tenantID, workspaceID, storageID, req)

	// Apply the resource to the cluster
	var resultCR *storagev1.Storage
	if isUpdate {
		// Update existing resource
		storageCR.ResourceVersion = existingStorage.ResourceVersion
		resultCR, err = c.updateBlockStorageCR(ctx, tenantID, storageCR)
		if err != nil {
			c.logger.ErrorContext(ctx, "failed to update block storage",
				slog.String("tenantID", tenantID),
				slog.String("storageID", storageID),
				slog.Any("error", err))
			return nil, false, fmt.Errorf("failed to update block storage '%s': %w", storageID, err)
		}
	} else {
		// Create new resource
		resultCR, err = c.createBlockStorageCR(ctx, tenantID, storageCR)
		if err != nil {
			c.logger.ErrorContext(ctx, "failed to create block storage",
				slog.String("tenantID", tenantID),
				slog.String("storageID", storageID),
				slog.Any("error", err))
			return nil, false, fmt.Errorf("failed to create block storage '%s': %w", storageID, err)
		}
	}

	// Convert the result back to SDK format
	sdkStorage := fromStorageCRToSDK(*resultCR)
	return &sdkStorage, isUpdate, nil
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

// Helper functions for BlockStorage CR operations

func (c StorageController) getBlockStorageCR(ctx context.Context, namespace, name string) (*storagev1.Storage, error) {
	uobj, err := c.client.Client.Resource(storagev1.StorageGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var storage storagev1.Storage
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(uobj.Object, &storage); err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to Storage: %w", err)
	}

	return &storage, nil
}

func (c StorageController) createBlockStorageCR(ctx context.Context, namespace string, storage *storagev1.Storage) (*storagev1.Storage, error) {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(storage)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Storage to unstructured: %w", err)
	}

	uobj := &unstructured.Unstructured{Object: unstructuredObj}
	created, err := c.client.Client.Resource(storagev1.StorageGVR).Namespace(namespace).Create(ctx, uobj, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	var result storagev1.Storage
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(created.Object, &result); err != nil {
		return nil, fmt.Errorf("failed to convert created unstructured to Storage: %w", err)
	}

	return &result, nil
}

func (c StorageController) updateBlockStorageCR(ctx context.Context, namespace string, storage *storagev1.Storage) (*storagev1.Storage, error) {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(storage)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Storage to unstructured: %w", err)
	}

	uobj := &unstructured.Unstructured{Object: unstructuredObj}
	updated, err := c.client.Client.Resource(storagev1.StorageGVR).Namespace(namespace).Update(ctx, uobj, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	var result storagev1.Storage
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(updated.Object, &result); err != nil {
		return nil, fmt.Errorf("failed to convert updated unstructured to Storage: %w", err)
	}

	return &result, nil
}

// Conversion functions between SDK and CR types

func fromSDKToStorageCR(tenantID, workspaceID, storageID string, req sdkschema.BlockStorage) *storagev1.Storage {
	storage := &storagev1.Storage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: storagev1.StorageGVR.GroupVersion().String(),
			Kind:       storagev1.StorageResource,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      storageID,
			Namespace: tenantID,
		},
		Spec: genv1.BlockStorageSpec{
			SizeGB: req.Spec.SizeGB,
			SkuRef: genv1.Reference{}, // todo add here something
		},
		RegionalComonSpec: regionalcommon.RegionalCommonSpec{
			Tenant:    tenantID,
			Workspace: workspaceID,
		},
		RegionalCommonData: regionalcommon.RegionalCommonData{
			Annotations: req.Annotations,
			Extensions:  req.Extensions,
			Labels:      req.Labels,
		},
	}

	// Handle optional SourceImageRef
	if req.Spec.SourceImageRef != nil {
		storage.Spec.SourceImageRef = &genv1.Reference{}
	}

	return storage
}

func fromStorageCRToSDK(cr storagev1.Storage) sdkschema.BlockStorage {
	storage := sdkschema.BlockStorage{
		Spec: sdkschema.BlockStorageSpec{
			SizeGB: cr.Spec.SizeGB,
			SkuRef: sdkschema.Reference{},
		},
		Annotations: cr.RegionalCommonData.Annotations,
		Extensions:  cr.RegionalCommonData.Extensions,
		Labels:      cr.RegionalCommonData.Labels,
	}

	// Handle optional SourceImageRef
	if cr.Spec.SourceImageRef != nil {
		storage.Spec.SourceImageRef = &sdkschema.Reference{}
	}

	// Populate metadata
	storage.Metadata = &sdkschema.RegionalWorkspaceResourceMetadata{
		Name:      cr.Name,
		Tenant:    cr.RegionalComonSpec.Tenant,
		Workspace: cr.RegionalComonSpec.Workspace,
	}

	// Handle status if present
	// Note: The status is typically populated by the controller, not by the API

	return storage
}
