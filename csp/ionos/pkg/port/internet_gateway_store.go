package port

import (
	"context"

	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

// InternetGatewayStore reconciles the SECA internet gateway. IONOS exposes no gateway object:
// internet access is a bool on the LAN (Lan.ForProvider.Public), which network_store.go always
// sets to true, so every network already has egress. Nothing maps to a gateway resource; this is
// a pure declaration, like RouteTableStore.
type InternetGatewayStore interface {
	Create(ctx context.Context, domain *internetgatewaydom.InternetGateway) error
	Delete(ctx context.Context, domain *internetgatewaydom.InternetGateway) error
}
