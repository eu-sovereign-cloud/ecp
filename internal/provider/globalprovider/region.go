package globalprovider

import (
	"context"
	"fmt"
	"log/slog"

	region "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/apis/regions/v1"
	"github.com/eu-sovereign-cloud/ecp/internal/kubeclient"
)

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

	return &RegionController{client: client,
		logger: logger.With(slog.String("component", "RegionController")),
	}, nil
}

// GetRegion retrieves a specific region by its ID by fetching the CR from the cluster.
func (c *RegionController) GetRegion(ctx context.Context, regionName string) (*region.Region, error) {
	gvr := schema.GroupVersionResource{
		Group:    regionsv1.Group,
		Version:  regionsv1.Version,
		Resource: regionsv1.Resource,
	}

	// Fetch the Regions custom resource from the Kubernetes API server. Cluster wide.
	unstructuredObj, err := c.client.Client.Resource(gvr).Get(ctx, regionName, metav1.GetOptions{})
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to get region CR", slog.String("region", regionName), slog.Any("error", err))
		return nil, fmt.Errorf("failed to retrieve region '%s': %w", regionName, err)
	}

	var crdRegion regionsv1.Regions
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &crdRegion); err != nil {
		c.logger.ErrorContext(ctx, "failed to convert unstructured object to Region type", slog.Any("error", err))
		return nil, fmt.Errorf("failed to convert unstructured object to Region type: %w", err)
	}

	// Convert the CRD spec to the SDK's RegionSpec type.
	providers := make([]region.Provider, len(crdRegion.Spec.Providers))
	for i, p := range crdRegion.Spec.Providers {
		providers[i] = region.Provider{
			Name:    p.Name,
			Url:     p.Url,
			Version: p.Version,
		}
	}

	sdkRegion := &region.Region{
		Spec: region.RegionSpec{
			AvailableZones: crdRegion.Spec.AvailableZones,
			Providers:      providers,
		},
		Metadata: &region.GlobalResourceMetadata{
			Kind: region.GlobalResourceMetadataKindRegion,
			Name: crdRegion.Name,
		},
	}

	return sdkRegion, nil
}

// ListRegions retrieves all available regions by listing the CRs from the cluster.
func (c *RegionController) ListRegions(ctx context.Context, params region.ListRegionsParams) (*secapi.Iterator[region.Region], error) {
	// Define the GroupVersionResource for the Region CRD.

	gvr := schema.GroupVersionResource{
		Group:    regionsv1.Group,
		Version:  regionsv1.Version,
		Resource: regionsv1.Resource,
	}

	// Fetch the list of Regions custom resources from the Kubernetes API server.
	unstructuredList, err := c.client.Client.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list region : %w", err)
	}

	sdkRegions := make([]region.Region, 0, len(unstructuredList.Items))
	for _, unstructuredObj := range unstructuredList.Items {
		var crdRegion regionsv1.Regions
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &crdRegion); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured object to Region type: %w", err)
		}

		// Convert the CRD spec to the SDK's RegionSpec type.
		providers := make([]region.Provider, len(crdRegion.Spec.Providers))
		for i, p := range crdRegion.Spec.Providers {
			providers[i] = region.Provider{
				Name:    p.Name,
				Url:     p.Url,
				Version: p.Version,
			}
		}

		sdkRegion := region.Region{
			Spec: region.RegionSpec{
				AvailableZones: crdRegion.Spec.AvailableZones,
				Providers:      providers,
			},
			Metadata: &region.GlobalResourceMetadata{
				Kind: region.GlobalResourceMetadataKindRegion,
				Name: crdRegion.Name,
			},
		}
		sdkRegions = append(sdkRegions, sdkRegion)
	}

	regionIterator := secapi.NewIterator(func(ctx context.Context, skipToken *string) ([]region.Region, *string, error) {
		return sdkRegions, skipToken, nil
	})
	return regionIterator, nil
}
