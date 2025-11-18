package region

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type GetRegion struct {
	Logger *slog.Logger
	Repo   port.ResourceQueryRepository[*model.RegionDomain]
}

// Do retrieves a specific region, maps it to the domain, and then projects it to the SDK model.
func (c *GetRegion) Do(ctx context.Context, name schema.ResourcePathParam) (*schema.Region, error) {
	regionDomain := &model.RegionDomain{
		Metadata: model.Metadata{Name: name},
	}
	err := c.Repo.Load(ctx, &regionDomain)
	if err != nil {
		return nil, err
	}

	sdkRegion := model.MapRegionDomainToSDK(*regionDomain, "get")

	return &sdkRegion, nil
}
