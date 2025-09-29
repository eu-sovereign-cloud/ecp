package regionalprovider

import (
    "context"
    "fmt"
    "log/slog"
    "strings"

    sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
    "github.com/eu-sovereign-cloud/go-sdk/secapi"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/client-go/rest"

    crdv1 "github.com/eu-sovereign-cloud/ecp/apis/storage/crds/v1"
    "github.com/eu-sovereign-cloud/ecp/internal/kubeclient"
    "github.com/eu-sovereign-cloud/ecp/internal/validation/filter"
)

type StorageSKUProvider interface {
	ListSKUs(ctx context.Context, tenantID string, params sdkstorage.ListSkusParams) (
		*secapi.Iterator[sdkstorage.StorageSku], error,
	)
	GetSKU(ctx context.Context, tenantID, skuID string) (*sdkstorage.StorageSku, error)
}

type ImageProvider interface {
	CreateOrUpdateImage(
		ctx context.Context, tenantID, imageID string,
		params sdkstorage.CreateOrUpdateImageParams, req sdkstorage.Image,
	) (*sdkstorage.Image, bool, error)
	GetImage(ctx context.Context, tenantID, imageID string) (*sdkstorage.Image, error)
	DeleteImage(ctx context.Context, tenantID, imageID string, params sdkstorage.DeleteImageParams) error
	ListImages(
		ctx context.Context, tenantID string, params sdkstorage.ListImagesParams,
	) (*secapi.Iterator[sdkstorage.Image], error)
}

type StorageProvider interface {
	StorageSKUProvider
	ImageProvider

	ListBlockStorages(
		ctx context.Context, tenantID, workspaceID string, params sdkstorage.ListBlockStoragesParams,
	) (*secapi.Iterator[sdkstorage.BlockStorage], error)
	GetBlockStorage(
		ctx context.Context, tenantID, workspaceID, storageID string,
	) (*sdkstorage.BlockStorage, error)
	CreateOrUpdateBlockStorage(
		ctx context.Context, tenantID, workspaceID, storageID string,
		params sdkstorage.CreateOrUpdateBlockStorageParams, req sdkstorage.BlockStorage,
	) (*sdkstorage.BlockStorage, bool, error)
	DeleteBlockStorage(
		ctx context.Context, tenantID, workspaceID, storageID string, params sdkstorage.DeleteBlockStorageParams,
	) error
}

var _ StorageProvider = (*StorageController)(nil) // Ensure StorageController implements the StorageProvider interface.

// StorageController implements the StorageProvider interface and provides methods to interact with the Storage CRDs and XRDs in the Kubernetes cluster.
type StorageController struct {
	client *kubeclient.KubeClient
	logger *slog.Logger
}

func (c StorageController) CreateOrUpdateImage(
	ctx context.Context, tenantID, imageID string, params sdkstorage.CreateOrUpdateImageParams, req sdkstorage.Image,
) (*sdkstorage.Image, bool, error) {
	// TODO implement me
	panic("implement me")
}

func (c StorageController) GetImage(ctx context.Context, tenantID, imageID string) (*sdkstorage.Image, error) {
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
) (*secapi.Iterator[sdkstorage.Image], error) {
	// TODO implement me
	panic("implement me")
}

// NewController creates a new StorageController with a Kubernetes client.
func NewController(logger *slog.Logger, cfg *rest.Config) (*StorageController, error) {
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

func (c StorageController) ListSKUs(
	ctx context.Context, tenantID string, params sdkstorage.ListSkusParams,
) (*secapi.Iterator[sdkstorage.StorageSku], error) {
	listOptions := metav1.ListOptions{}
	rawSelector := ""
	if params.Labels != nil {
		rawSelector = *params.Labels
		// Pass a subset of simple "key=value" selectors to the API for pre-filtering.
		listOptions.LabelSelector = filter.K8sSelectorForAPI(rawSelector)
	}

	// Always filter by tenant ID to ensure tenant isolation, since we cannot use namespaces (yet).
	if listOptions.LabelSelector != "" {
		listOptions.LabelSelector = strings.Join(
			[]string{
				fmt.Sprintf("%s=%s", tenantLabelKey, tenantID), listOptions.LabelSelector,
			}, ",",
		)
	} else {
		listOptions.LabelSelector = fmt.Sprintf("%s=%s", tenantLabelKey, tenantID)
	}

	unstructuredList, err := c.client.Client.Resource(crdv1.StorageSKUGVR).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage SKU CRs: %w", err)
	}

	sdkStorageSKUs := make([]sdkstorage.StorageSku, 0, len(unstructuredList.Items))
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

		var crdStorageSKU crdv1.StorageSKU
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
			unstructuredObj.Object, &crdStorageSKU,
		); err != nil {
			c.logger.ErrorContext(
				ctx, "failed to convert unstructured object to StorageSKU type", slog.Any("error", err),
			)
			return nil, fmt.Errorf("failed to convert unstructured object to StorageSKU type: %w", err)
		}

		sdkStorageSKU, err := fromCRToSDKStorageSKU(crdStorageSKU)
		if err != nil {
			c.logger.ErrorContext(ctx, "failed to convert CR to SDK storage SKU", slog.Any("error", err))
			return nil, fmt.Errorf("failed to convert CR to SDK storage SKU: %w", err)
		}
		sdkStorageSKUs = append(sdkStorageSKUs, sdkStorageSKU)
	}

	skuIterator := secapi.NewIterator(
		func(ctx context.Context, skipToken *string) ([]sdkstorage.StorageSku, *string, error) {
			return sdkStorageSKUs, nil, nil
		},
	)
	return skuIterator, nil
}

func (c StorageController) GetSKU(
	ctx context.Context, tenantID, skuID string,
) (*sdkstorage.StorageSku, error) {
	skuName := fmt.Sprintf(tenantWideResourceNamePattern, tenantID, skuID)

	// Fetch the Storage SKU custom resource from the Kubernetes API server. Cluster wide.
	unstructuredObj, err := c.client.Client.Resource(crdv1.StorageSKUGVR).Get(
		ctx, skuName, metav1.GetOptions{},
	)
	if err != nil {
		c.logger.ErrorContext(
			ctx, "failed to get storage SKU CR", slog.String("sku", skuID), slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to retrieve storage SKU '%s': %w", skuID, err)
	}

	var crdStorageSKU crdv1.StorageSKU
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
		unstructuredObj.Object, &crdStorageSKU,
	); err != nil {
		c.logger.ErrorContext(
			ctx, "failed to convert unstructured object to StorageSKU type", slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to convert unstructured object to StorageSKU type: %w", err)
	}

	// Convert the CR spec to the SDK's StorageSku type.
	sdkStorageSKU, err := fromCRToSDKStorageSKU(crdStorageSKU)
	if err != nil {
		c.logger.ErrorContext(
			ctx, "failed to convert CR to SDK storage SKU", slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to convert CR to SDK storage SKU: %w", err)
	}

	return &sdkStorageSKU, nil
}

func (c StorageController) ListBlockStorages(
	ctx context.Context, tenantID, workspaceID string, params sdkstorage.ListBlockStoragesParams,
) (*secapi.Iterator[sdkstorage.BlockStorage], error) {
	// TODO implement me
	panic("implement me")
}

func (c StorageController) GetBlockStorage(
	ctx context.Context, tenantID, workspaceID, storageID string,
) (*sdkstorage.BlockStorage, error) {
	// TODO implement me
	panic("implement me")
}

func (c StorageController) CreateOrUpdateBlockStorage(
	ctx context.Context, tenantID, workspaceID, storageID string,
	params sdkstorage.CreateOrUpdateBlockStorageParams, req sdkstorage.BlockStorage,
) (*sdkstorage.BlockStorage, bool, error) {
    // TODO implement me
    panic("implement me")
}

func (c StorageController) DeleteBlockStorage(
	ctx context.Context, tenantID, workspaceID, storageID string, params sdkstorage.DeleteBlockStorageParams,
) error {
	// TODO implement me
	panic("implement me")
}

func fromCRToSDKStorageSKU(crStorageSKU crdv1.StorageSKU) (sdkstorage.StorageSku, error) {
	sdkStorageSKU := sdkstorage.StorageSku{
		Spec: &sdkstorage.StorageSkuSpec{
			Iops:          crStorageSKU.Spec.Iops,
			MinVolumeSize: crStorageSKU.Spec.MinVolumeSize,
			Type:          sdkstorage.StorageSkuSpecType(crStorageSKU.Spec.Type),
		},
		Metadata: &sdkstorage.SkuResourceMetadata{
			Name: crStorageSKU.GetName(),
		},
	}
	return sdkStorageSKU, nil
}
