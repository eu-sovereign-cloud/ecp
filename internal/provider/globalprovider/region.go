package globalprovider

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"

	region "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"k8s.io/client-go/rest"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/apis/regions/v1"
	"github.com/eu-sovereign-cloud/ecp/internal/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/internal/provider/common"
	"github.com/eu-sovereign-cloud/ecp/internal/validation"
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
	convert := common.Adapter(func(crdRegion regionsv1.Region) (schema.Region, error) {
		return fromCRDToSDKRegion(crdRegion, "get")
	})
	opts := common.NewGetOptions()
	regionObj, err := common.GetResource(ctx, c.client.Client, regionsv1.GroupVersionResource, regionName, c.logger, convert, opts)
	if err != nil {
		return nil, err
	}
	return &regionObj, nil
}

// ListRegions retrieves all available regions by listing the CRs from the cluster.
func (c *RegionController) ListRegions(ctx context.Context, params region.ListRegionsParams) (*region.RegionIterator, error) {
	limit := validation.GetLimit(params.Limit)
	convert := common.Adapter(func(crdRegion regionsv1.Region) (schema.Region, error) {
		return fromCRDToSDKRegion(crdRegion, "list")
	})
	opts := common.NewListOptions()
	if limit > 0 {
		opts.Limit(limit)
	}
	if params.SkipToken != nil {
		opts.SkipToken(*params.SkipToken)
	}
	if params.Labels != nil {
		opts.Selector(*params.Labels)
	}

	sdkRegions, nextSkipToken, err := common.ListResources(ctx, c.client.Client, regionsv1.GroupVersionResource, c.logger, convert, opts)
	if err != nil {
		return nil, err
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
	resource, err := url.JoinPath(RegionBaseURL, regionsv1.Resource, crdRegion.Name)
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
