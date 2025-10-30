package regionalprovider

import (
	"context"
	"fmt"
	"log/slog"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	skuv1 "github.com/eu-sovereign-cloud/ecp/apis/block-storage/skus/v1"
	"github.com/eu-sovereign-cloud/ecp/internal/validation"

	"github.com/eu-sovereign-cloud/ecp/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/internal/validation/filter"
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
	listOptions := metav1.ListOptions{
		Limit: int64(limit),
	}
	if params.SkipToken != nil {
		listOptions.Continue = *params.SkipToken
	}

	rawSelector := ""
	if params.Labels != nil {
		rawSelector = *params.Labels
		// Pass a subset of simple "key=value" selectors to the API for pre-filtering.
		listOptions.LabelSelector = filter.K8sSelectorForAPI(rawSelector)
	}

	unstructuredList, err := c.client.Client.Resource(skuv1.StorageSKUGVR).Namespace(tenantID).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage SKU CRs: %w", err)
	}

	sdkStorageSKUs := make([]sdkschema.StorageSku, 0, len(unstructuredList.Items))
	for _, unstructuredObj := range unstructuredList.Items {
		// Apply the full, custom client-side filter for numeric and wildcards
		if rawSelector != "" {
			match, k8sHandled, err := filter.MatchLabels(unstructuredObj.GetLabels(), rawSelector)
			if err != nil {
				c.logger.WarnContext(
					ctx, "skipping resource due to invalid label selector", "error", err, "selector", rawSelector,
				)
				continue
			}
			if !match && !k8sHandled {
				continue
			}
		}

		var crdStorageSKU skuv1.StorageSKU
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
			unstructuredObj.Object, &crdStorageSKU,
		); err != nil {
			c.logger.ErrorContext(
				ctx, "failed to convert unstructured object to StorageSKU type", slog.Any("error", err),
			)
			return nil, fmt.Errorf("failed to convert unstructured object to StorageSKU type: %w", err)
		}

		sdkStorageSKUs = append(sdkStorageSKUs, fromCRToSDKStorageSKU(crdStorageSKU))
	}

	iterator := sdkstorage.SkuIterator{
		Items: sdkStorageSKUs,
		Metadata: sdkschema.ResponseMetadata{
			Provider:  ProviderStorageName,
			Resource:  skuv1.StorageSKUResource,
			SkipToken: nil,
			Verb:      "list",
		},
	}
	nextSkipToken := unstructuredList.GetContinue()
	if nextSkipToken != "" {
		iterator.Metadata.SkipToken = &nextSkipToken
	}
	return &iterator, nil
}

func (c StorageController) GetSKU(
	ctx context.Context, tenantID, skuID string,
) (*sdkschema.StorageSku, error) {
	// TODO - add tenant support once it's implemented
	// Fetch the Storage SKU custom resource from the Kubernetes API server. Cluster wide.
	unstructuredObj, err := c.client.Client.Resource(skuv1.StorageSKUGVR).Namespace(tenantID).Get(ctx, skuID, metav1.GetOptions{})
	if err != nil {
		c.logger.ErrorContext(
			ctx, "failed to get storage SKU CR", slog.String("sku", skuID), slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to retrieve storage SKU '%s': %w", skuID, err)
	}

	var crdStorageSKU skuv1.StorageSKU
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
		unstructuredObj.Object, &crdStorageSKU,
	); err != nil {
		c.logger.ErrorContext(
			ctx, "failed to convert unstructured object to StorageSKU type", slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to convert unstructured object to StorageSKU type: %w", err)
	}

	// Convert the CR spec to the SDK's StorageSku type.
	sdkStorageSKU := fromCRToSDKStorageSKU(crdStorageSKU)
	return &sdkStorageSKU, nil
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
