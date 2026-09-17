package internetgateway

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

type CreateInternetGateway struct {
	Store port.InternetGatewayStore
}

func (c *CreateInternetGateway) Do(ctx context.Context, domain *internetgatewaydom.InternetGateway) error {
	return c.Store.Create(ctx, domain)
}
