package port

import (
	"context"

	netdom "github.com/eu-sovereign-cloud/ecp/resources/network/networks/v1"
)

type NetworkStore interface {
	Create(ctx context.Context, domain *netdom.Network) error
	Delete(ctx context.Context, domain *netdom.Network) error
}
