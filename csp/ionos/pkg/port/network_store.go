package port

import (
	"context"

	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

type NetworkStore interface {
	Create(ctx context.Context, domain *netdom.Network) error
	Delete(ctx context.Context, domain *netdom.Network) error
}
