package kubernetes

import (
	"context"

	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

// SubnetPlugin is implemented by CSP plugins that manage Subnet resources.
type SubnetPlugin interface {
	Create(ctx context.Context, resource *subnetdom.Subnet) error
	Delete(ctx context.Context, resource *subnetdom.Subnet) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *subnetdom.Subnet) error
}
