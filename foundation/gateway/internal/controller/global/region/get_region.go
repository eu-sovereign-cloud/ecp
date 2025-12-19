package region

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type GetRegion struct {
	Logger *slog.Logger
	Repo   port.ReaderRepo[*model.RegionDomain]
}

// Do retrieves a specific region, maps it to the domain, and then projects it to the SDK model.
func (c *GetRegion) Do(ctx context.Context, ir port.IdentifiableResource) (*model.RegionDomain, error) {
	regionDomain := &model.RegionDomain{
		Metadata: model.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: ir.GetName(),
			},
		},
	}
	err := c.Repo.Load(ctx, &regionDomain)
	if err != nil {
		return nil, err
	}

	return regionDomain, nil
}
