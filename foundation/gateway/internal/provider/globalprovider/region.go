package globalprovider

import (
	"context"
	"log/slog"

	region "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

var _ RegionProvider = (*RegionController)(nil) // Ensure RegionController implements the RegionProvider interface.

// RegionProvider defines the interface for interacting with regions in the ECP.
type RegionProvider interface {
	GetRegion(ctx context.Context, name string) (*schema.Region, error)
	ListRegions(ctx context.Context, params region.ListRegionsParams) (*region.RegionIterator, error)
}

// RegionController implements the RegionalProvider interface
type RegionController struct {
	logger     *slog.Logger
	regionRepo port.ResourceQueryRepository[*model.RegionDomain]
}

// NewController creates a new RegionController.
func NewController(
	logger *slog.Logger,
	regionRepo port.ResourceQueryRepository[*model.RegionDomain],
) *RegionController {
	return &RegionController{
		logger:     logger.With(slog.String("component", "RegionController")),
		regionRepo: regionRepo,
	}
}

// GetRegion retrieves a specific region, maps it to the domain, and then projects it to the SDK model.
func (c *RegionController) GetRegion(ctx context.Context, regionName schema.ResourcePathParam) (*schema.Region, error) {
	regionDomain := &model.RegionDomain{
		Metadata: model.Metadata{Name: regionName},
	}
	err := c.regionRepo.Load(ctx, &regionDomain)
	if err != nil {
		return nil, err
	}

	sdkRegion := model.MapRegionDomainToSDK(*regionDomain, "get")

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

	listParams := model.ListParams{
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}

	var domainRegions []*model.RegionDomain
	nextSkipToken, err := c.regionRepo.List(ctx, listParams, &domainRegions)
	if err != nil {
		return nil, err
	}

	sdkRegions := make([]schema.Region, len(domainRegions))
	for i, dom := range domainRegions {
		sdkRegions[i] = model.MapRegionDomainToSDK(*dom, "list")
	}

	iterator := &region.RegionIterator{
		Items: sdkRegions,
		Metadata: schema.ResponseMetadata{
			Provider: model.ProviderRegionName,
			Resource: regionsv1.Resource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return iterator, nil
}
