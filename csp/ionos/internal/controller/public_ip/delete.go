package publicip

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

type DeletePublicIP struct {
	Store port.PublicIPStore
}

func (c *DeletePublicIP) Do(ctx context.Context, domain *publicipdom.PublicIp) error {
	return c.Store.Delete(ctx, domain)
}
