package kubernetes

import (
	"context"

	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// InstancePlugin is implemented by CSP plugins that manage Instance resources.
type InstancePlugin interface {
	Create(ctx context.Context, resource *instancedom.Instance) error
	Delete(ctx context.Context, resource *instancedom.Instance) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *instancedom.Instance) error
	// PowerOn transitions an instance to the powered-on state.
	PowerOn(ctx context.Context, resource *instancedom.Instance) error
	// PowerOff transitions an instance to the powered-off state.
	PowerOff(ctx context.Context, resource *instancedom.Instance) error
}
