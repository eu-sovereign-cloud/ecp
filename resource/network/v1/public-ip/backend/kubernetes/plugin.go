package kubernetes

import (
	"context"

	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

// PublicIpPlugin is implemented by CSP plugins that manage PublicIp resources.
type PublicIpPlugin interface {
	Create(ctx context.Context, resource *publicipdom.PublicIp) error
	Delete(ctx context.Context, resource *publicipdom.PublicIp) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *publicipdom.PublicIp) error
}
