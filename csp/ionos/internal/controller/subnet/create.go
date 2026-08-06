package subnet

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

type CreateSubnet struct {
	Store port.SubnetStore
}

func (c *CreateSubnet) Do(ctx context.Context, domain *subnetdom.Subnet) error {
	return c.Store.Create(ctx, domain)
}
