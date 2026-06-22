package network

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/pkg/port"
)

type DeleteNetwork struct {
	Store port.NetworkStore
}

func (d *DeleteNetwork) Do(ctx context.Context, domain *regional.NetworkDomain) error {
	return d.Store.Delete(ctx, domain)
}
