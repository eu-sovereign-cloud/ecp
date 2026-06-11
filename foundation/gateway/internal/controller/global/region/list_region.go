package region

import (
	"context"
	"log/slog"

	model "github.com/eu-sovereign-cloud/ecp/foundation/models"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

type ListRegion struct {
	Logger *slog.Logger
	Repo   port.ReaderRepo[*model.RegionDomain]
}

// Do retrieves all available regions, maps them to the domain, and then projects them to the SDK model.
func (c *ListRegion) Do(ctx context.Context, params model.ListParams) ([]*model.RegionDomain, *string, error) {
	var domainRegions []*model.RegionDomain
	nextSkipToken, err := c.Repo.List(ctx, params, &domainRegions)
	if err != nil {
		return nil, nil, err
	}

	return domainRegions, nextSkipToken, nil
}
