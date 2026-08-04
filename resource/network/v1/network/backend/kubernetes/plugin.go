package kubernetes

import (
	"context"

	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

// NetworkPlugin is implemented by CSP plugins that manage network resources.
type NetworkPlugin interface {
	Create(ctx context.Context, resource *netdom.Network) error
	Delete(ctx context.Context, resource *netdom.Network) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *netdom.Network) error
}
