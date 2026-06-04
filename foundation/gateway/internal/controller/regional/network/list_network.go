package network

import (
	"context"
	"log/slog"

	model "github.com/eu-sovereign-cloud/ecp/foundation/models/domain"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

// ListNetworks lists network resources.
type ListNetworks struct {
	Logger      *slog.Logger
	NetworkRepo port.ReaderRepo[*regional.NetworkDomain]
}

func (c ListNetworks) Do(ctx context.Context, params model.ListParams) (
	[]*regional.NetworkDomain, *string, error,
) {
	var domainNetworks []*regional.NetworkDomain
	nextSkipToken, err := c.NetworkRepo.List(ctx, params, &domainNetworks)
	if err != nil {
		return nil, nil, err
	}

	return domainNetworks, nextSkipToken, nil
}
