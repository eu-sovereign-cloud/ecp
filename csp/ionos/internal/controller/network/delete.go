package network

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

type DeleteNetwork struct {
	Store port.NetworkStore
}

func (d *DeleteNetwork) Do(ctx context.Context, domain *netdom.Network) error {
	return d.Store.Delete(ctx, domain)
}
