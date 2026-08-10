package kubernetes

import (
	"context"

	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

// InternetGatewayPlugin is implemented by CSP plugins that manage InternetGateway resources.
type InternetGatewayPlugin interface {
	Create(ctx context.Context, resource *internetgatewaydom.InternetGateway) error
	Delete(ctx context.Context, resource *internetgatewaydom.InternetGateway) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *internetgatewaydom.InternetGateway) error
}
