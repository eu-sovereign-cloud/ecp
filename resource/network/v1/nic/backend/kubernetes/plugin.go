package kubernetes

import (
	"context"

	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

// NicPlugin is implemented by CSP plugins that manage NIC resources.
type NicPlugin interface {
	Create(ctx context.Context, resource *nicdom.Nic) error
	Delete(ctx context.Context, resource *nicdom.Nic) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *nicdom.Nic) error
}
