package kubernetes

import (
	"context"

	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

// InternetGatewayPlugin is implemented by CSP plugins that manage InternetGateway resources.
type InternetGatewayPlugin interface {
	Create(ctx context.Context, resource *internetgatewaydom.InternetGateway) error
	Delete(ctx context.Context, resource *internetgatewaydom.InternetGateway) error
}
