package globalprovider

import (
	"context"
	"fmt"
	"log/slog"

	region "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"k8s.io/client-go/rest"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/port"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/provider/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
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
	logger     *slog.Logger
	regionRepo port.ResourceQueryRepository[model.RegionDomain]
}

// NewController creates a new RegionController with a Kubernetes client.
func NewController(logger *slog.Logger, cfg *rest.Config) (*RegionController, error) {
	client, err := kubeclient.NewFromConfig(cfg)
	if err != nil {
		logger.Error("failed to create kubeclient", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create kubeclient: %w", err)
	}

	// This converter now maps from the K8s unstructured object to the internal domain model.
	crdToDomainConverter := func(u unstructured.Unstructured) (model.RegionDomain, error) {
		var crdRegion regionsv1.Region
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &crdRegion); err != nil {
			return model.RegionDomain{}, err
		}
		return model.MapRegionCRDToDomain(crdRegion)
	}

	regionAdapter := kubernetes.NewAdapter(
		client.Client,
		regionsv1.GroupVersionResource,
		logger,
		crdToDomainConverter,
	)

	return &RegionController{
		logger:     logger.With(slog.String("component", "RegionController")),
		regionRepo: regionAdapter,
	}, nil
}

// GetRegion retrieves a specific region, maps it to the domain, and then projects it to the SDK model.
func (c *RegionController) GetRegion(ctx context.Context, regionName schema.ResourcePathParam) (*schema.Region, error) {
	regionDomain, err := c.regionRepo.Get(ctx, "", regionName)
	if err != nil {
		return nil, err
	}

	sdkRegion := model.MapRegionDomainToSDK(regionDomain, "get")

	return &sdkRegion, nil
}

// ListRegions retrieves all available regions, maps them to the domain, and then projects them to the SDK model.
func (c *RegionController) ListRegions(ctx context.Context, params region.ListRegionsParams) (*region.RegionIterator, error) {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	listParams := port.ListParams{
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}

	domainRegions, nextSkipToken, err := c.regionRepo.List(ctx, listParams)
	if err != nil {
		return nil, err
	}

	sdkRegions := make([]schema.Region, len(domainRegions))
	for i, dom := range domainRegions {
		sdkRegions[i] = model.MapRegionDomainToSDK(dom, "list")
	}

	iterator := &region.RegionIterator{
		Items: sdkRegions,
		Metadata: schema.ResponseMetadata{
			Provider: ProviderRegionName,
			Resource: regionsv1.Resource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return iterator, nil
}
