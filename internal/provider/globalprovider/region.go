package globalprovider

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"

	region "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/apis/regions/v1"
	"github.com/eu-sovereign-cloud/ecp/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/internal/validation/filter"
)

const (
	RegionBaseURL      = "/providers/seca.region"
	ProviderRegionName = "seca.region/v1"
)

var _ RegionProvider = (*RegionController)(nil) // Ensure RegionController implements the RegionProvider interface.

// RegionProvider defines the interface for interacting with regions in the ECP.
type RegionProvider interface {
	GetRegion(ctx context.Context, name string) (*schema.Region, error)
	ListRegions(ctx context.Context, params region.ListRegionsParams) (*region.RegionIterator, error)
}

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
func (c *RegionController) GetRegion(ctx context.Context, regionName schema.ResourcePathParam) (*schema.Region, error) {
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
	sdkRegion, err := fromCRDToSDKRegion(crdRegion, "get")
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to convert CR to SDK region", slog.Any("error", err))
		return nil, fmt.Errorf("failed to convert CR to SDK region: %w", err)
	}

	return &sdkRegion, nil
}

// ListRegions retrieves all available regions by listing the CRs from the cluster.
func (c *RegionController) ListRegions(ctx context.Context, params region.ListRegionsParams) (*region.RegionIterator, error) {
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

	unstructuredList, err := c.client.Client.Resource(regionsv1.GroupVersionResource).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list region CRs: %w", err)
	}

	sdkRegions := make([]schema.Region, 0, len(unstructuredList.Items))
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

		sdkRegion, err := fromCRDToSDKRegion(crdRegion, "list")
		if err != nil {
			c.logger.ErrorContext(ctx, "failed to convert CR to SDK region", slog.Any("error", err))
			return nil, fmt.Errorf("failed to convert CR to SDK region: %w", err)
		}
		sdkRegions = append(sdkRegions, sdkRegion)
	}

	iterator := &region.RegionIterator{
		Items: sdkRegions,
		Metadata: schema.ResponseMetadata{
			Provider: ProviderRegionName,
			Resource: "regions",
			Verb:     "list",
		},
	}

	nextSkipToken := unstructuredList.GetContinue()
	if nextSkipToken != "" {
		iterator.Metadata.SkipToken = &nextSkipToken
	}

	return iterator, nil
}

func fromCRDToSDKRegion(crdRegion regionsv1.Region, verb string) (schema.Region, error) {
	providers := make([]schema.Provider, len(crdRegion.Spec.Providers))
	for i, provider := range crdRegion.Spec.Providers {
		providers[i] = schema.Provider{
			Name:    provider.Name,
			Url:     provider.Url,
			Version: provider.Version,
		}
	}
	resVersion, err := strconv.Atoi(crdRegion.GetResourceVersion())
	if err != nil {
		return schema.Region{}, fmt.Errorf("could not parse resource version: %w", err)
	}
	resource, err := url.JoinPath(RegionBaseURL, "regions", crdRegion.Name)
	if err != nil {
		return schema.Region{}, fmt.Errorf("could not parse resource URL: %w", err)
	}
	refObj := schema.ReferenceObject{
		Resource: resource,
	}
	reference := schema.Reference{}
	if err := reference.FromReferenceObject(refObj); err != nil {
		return schema.Region{}, fmt.Errorf("could not convert to reference object: %w", err)
	}

	sdkRegion := schema.Region{
		Spec: schema.RegionSpec{
			AvailableZones: crdRegion.Spec.AvailableZones,
			Providers:      providers,
		},
		Metadata: &schema.GlobalResourceMetadata{
			ApiVersion:      regionsv1.Version,
			CreatedAt:       crdRegion.GetCreationTimestamp().Time,
			LastModifiedAt:  crdRegion.GetCreationTimestamp().Time,
			Kind:            schema.GlobalResourceMetadataKindResourceKindRegion,
			Name:            crdRegion.GetName(),
			Provider:        ProviderRegionName,
			Resource:        crdRegion.GetName(),
			Ref:             &reference,
			ResourceVersion: resVersion,
			Verb:            verb,
		},
	}
	if crdRegion.GetDeletionTimestamp() != nil {
		sdkRegion.Metadata.DeletedAt = &crdRegion.GetDeletionTimestamp().Time
	}
	return sdkRegion, nil
}
