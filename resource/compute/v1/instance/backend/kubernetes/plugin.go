package kubernetes

import (
	"context"

	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// InstancePlugin is implemented by CSP plugins that manage Instance resources.
type InstancePlugin interface {
	Create(ctx context.Context, resource *instancedom.Instance) error
	Delete(ctx context.Context, resource *instancedom.Instance) error
}
