package network

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
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
