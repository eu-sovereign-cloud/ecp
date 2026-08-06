package subnet

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

type DeleteSubnet struct {
	Store port.SubnetStore
}

func (c *DeleteSubnet) Do(ctx context.Context, domain *subnetdom.Subnet) error {
	return c.Store.Delete(ctx, domain)
}
