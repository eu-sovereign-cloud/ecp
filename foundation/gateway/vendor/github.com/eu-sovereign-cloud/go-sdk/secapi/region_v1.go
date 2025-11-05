package secapi

import (
	"context"
	"net/http"

	region "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"k8s.io/utils/ptr"
)

type RegionV1 struct {
	API
	region region.ClientWithResponsesInterface
}

// Region

func (api *RegionV1) ListRegions(ctx context.Context) (*Iterator[schema.Region], error) {
	iter := Iterator[schema.Region]{
		fn: func(ctx context.Context, skipToken *string) ([]schema.Region, *string, error) {
			resp, err := api.region.ListRegionsWithResponse(ctx, &region.ListRegionsParams{
				Accept: ptr.To(region.ListRegionsParamsAccept(schema.AcceptHeaderJson)),
			}, api.loadRequestHeaders)
			if err != nil {
				return nil, nil, err
			}

			return resp.JSON200.Items, resp.JSON200.Metadata.SkipToken, nil
		},
	}

	return &iter, nil
}

func (api *RegionV1) GetRegion(ctx context.Context, name string) (*schema.Region, error) {
	resp, err := api.region.GetRegionWithResponse(ctx, name, api.loadRequestHeaders)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrResourceNotFound
	} else {
		return resp.JSON200, nil
	}
}

func newRegionV1(client *GlobalClient, regionsUrl string) (*RegionV1, error) {
	region, err := region.NewClientWithResponses(regionsUrl)
	if err != nil {
		return nil, err
	}

	return &RegionV1{API: API{authToken: client.authToken}, region: region}, nil
}
