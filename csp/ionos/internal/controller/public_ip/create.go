package publicip

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

type CreatePublicIP struct {
	Store port.PublicIPStore
}

func (c *CreatePublicIP) Do(ctx context.Context, domain *publicipdom.PublicIp) error {
	return c.Store.Create(ctx, domain)
}
