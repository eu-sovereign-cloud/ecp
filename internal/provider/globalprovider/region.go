package globalprovider

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	region "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/apis/regions/v1"
	"github.com/eu-sovereign-cloud/ecp/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/internal/validation/filter"
)

const ProviderRegionName = "seca.region/v1"

// RegionProvider defines the interface for interacting with regions in the ECP.
type RegionProvider interface {
	GetRegion(ctx context.Context, name string) (*region.Region, error)
	ListRegions(ctx context.Context, params region.ListRegionsParams) (*secapi.Iterator[region.Region], error)
}

var _ RegionProvider = (*RegionController)(nil) // Ensure RegionController implements the RegionProvider interface.

// RegionController implements the RegionalProvider interface and provides methods to interact with the Region CRD in the Kubernetes cluster.
type RegionController struct {
	client *kubeclient.KubeClient
	logger *slog.Logger
}

// NewController creates a new RegionController with a Kubernetes client.
func NewController(logger *slog.Logger, cfg *rest.Config) (*RegionController, error) {
	client, err := kubeclient.NewFromConfig(cfg)
	if err != nil {
		logger.Error("failed to create kubeclient", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create kubeclient: %w", err)
	}

	return &RegionController{
		client: client,
		logger: logger.With(slog.String("component", "RegionController")),
	}, nil
}

// GetRegion retrieves a specific region by its ID by fetching the CR from the cluster.
func (c *RegionController) GetRegion(ctx context.Context, regionName string) (*region.Region, error) {
	// Fetch the Regions custom resource from the Kubernetes API server. Cluster wide.
	unstructuredObj, err := c.client.Client.Resource(regionsv1.GroupVersionResource).Get(ctx, regionName, metav1.GetOptions{})
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to get region CR", slog.String("region", regionName), slog.Any("error", err))
		return nil, fmt.Errorf("failed to retrieve region '%s': %w", regionName, err)
	}

	var crdRegion regionsv1.Region
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &crdRegion); err != nil {
		c.logger.ErrorContext(ctx, "failed to convert unstructured object to Region type", slog.Any("error", err))
		return nil, fmt.Errorf("failed to convert unstructured object to Region type: %w", err)
	}

	// Convert the CR spec to the SDK's RegionSpec type.
	sdkRegion, err := fromCRToSDKRegion(crdRegion, "get")
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to convert CR to SDK region", slog.Any("error", err))
		return nil, fmt.Errorf("failed to convert CR to SDK region: %w", err)
	}

	return &sdkRegion, nil
}

// ListRegions retrieves all available regions by listing the CRs from the cluster.
func (c *RegionController) ListRegions(ctx context.Context, params region.ListRegionsParams) (*secapi.Iterator[region.Region], error) {
	listOptions := metav1.ListOptions{}
	rawSelector := ""
	if params.Labels != nil {
		rawSelector = *params.Labels
		// Pass a subset of simple "key=value" selectors to the API for pre-filtering.
		listOptions.LabelSelector = filter.K8sSelectorForAPI(rawSelector)
	}

	unstructuredList, err := c.client.Client.Resource(regionsv1.GroupVersionResource).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list region CRs: %w", err)
	}

	sdkRegions := make([]region.Region, 0, len(unstructuredList.Items))
	for _, unstructuredObj := range unstructuredList.Items {
		// Apply the full, custom client-side filter for numeric and wildcards
		if rawSelector != "" {
			match, k8sHandled, err := filter.MatchLabels(unstructuredObj.GetLabels(), rawSelector)
			if err != nil {
				c.logger.WarnContext(ctx, "skipping resource due to invalid label selector", "error", err, "selector", rawSelector)
				continue
			}
			if !match && !k8sHandled {
				continue
			}
		}

		var crdRegion regionsv1.Region
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &crdRegion); err != nil {
			c.logger.ErrorContext(ctx, "failed to convert unstructured object to Region type", slog.Any("error", err))
			return nil, fmt.Errorf("failed to convert unstructured object to Region type: %w", err)
		}

		sdkRegion, err := fromCRToSDKRegion(crdRegion, "list")
		if err != nil {
			c.logger.ErrorContext(ctx, "failed to convert CR to SDK region", slog.Any("error", err))
			return nil, fmt.Errorf("failed to convert CR to SDK region: %w", err)
		}
		sdkRegions = append(sdkRegions, sdkRegion)
	}

	regionIterator := secapi.NewIterator(func(ctx context.Context, skipToken *string) ([]region.Region, *string, error) {
		return sdkRegions, nil, nil
	})
	return regionIterator, nil
}

func fromCRToSDKRegion(crRegion regionsv1.Region, verb string) (region.Region, error) {
	providers := make([]region.Provider, len(crRegion.Spec.Providers))
	for i, provider := range crRegion.Spec.Providers {
		providers[i] = region.Provider{
			Name:    provider.Name,
			Url:     provider.Url,
			Version: provider.Version,
		}
	}
	resVersion, err := strconv.Atoi(crRegion.GetResourceVersion())
	if err != nil {
		return region.Region{}, fmt.Errorf("could not parse resource version: %w", err)
	}
	refObj := region.ReferenceObject{
		Resource: "regions/" + crRegion.Name,
	}
	reference := region.Reference{}
	if err := reference.FromReferenceObject(refObj); err != nil {
		return region.Region{}, fmt.Errorf("could not convert to reference object: %w", err)
	}

	sdkRegion := region.Region{
		Spec: region.RegionSpec{
			AvailableZones: crRegion.Spec.AvailableZones,
			Providers:      providers,
		},
		Metadata: &region.GlobalResourceMetadata{
			ApiVersion:      regionsv1.Version,
			CreatedAt:       crRegion.GetCreationTimestamp().Time,
			LastModifiedAt:  crRegion.GetCreationTimestamp().Time,
			Kind:            region.GlobalResourceMetadataKindRegion,
			Name:            crRegion.GetName(),
			Provider:        ProviderRegionName,
			Resource:        crRegion.GetName(),
			Ref:             &reference,
			ResourceVersion: resVersion,
			Verb:            verb,
		},
	}
	if crRegion.GetDeletionTimestamp() != nil {
		sdkRegion.Metadata.DeletedAt = &crRegion.GetDeletionTimestamp().Time
	}
	return sdkRegion, nil
}
