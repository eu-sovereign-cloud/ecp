package network

import (
	"context"

	netdom "github.com/eu-sovereign-cloud/ecp/resources/regional/network/networks/v1/domain"
	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
)

type DeleteNetwork struct {
	Store port.NetworkStore
}

func (d *DeleteNetwork) Do(ctx context.Context, domain *netdom.NetworkDomain) error {
	return d.Store.Delete(ctx, domain)
}
