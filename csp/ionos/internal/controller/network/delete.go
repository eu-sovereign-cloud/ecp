package network

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
)

type DeleteNetwork struct {
	Store port.NetworkStore
}

func (d *DeleteNetwork) Do(ctx context.Context, domain *regional.NetworkDomain) error {
	return d.Store.Delete(ctx, domain)
}
