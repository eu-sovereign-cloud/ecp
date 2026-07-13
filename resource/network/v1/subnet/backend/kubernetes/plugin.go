package kubernetes

import (
	"context"

	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

// SubnetPlugin is implemented by CSP plugins that manage Subnet resources.
type SubnetPlugin interface {
	Create(ctx context.Context, resource *subnetdom.Subnet) error
	Delete(ctx context.Context, resource *subnetdom.Subnet) error
}
