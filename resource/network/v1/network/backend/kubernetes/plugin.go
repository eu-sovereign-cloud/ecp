package kubernetes

import (
	"context"

	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

// NetworkPlugin is implemented by CSP plugins that manage network resources.
type NetworkPlugin interface {
	Create(ctx context.Context, resource *netdom.Network) error
	Delete(ctx context.Context, resource *netdom.Network) error
}
