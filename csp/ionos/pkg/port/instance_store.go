package port

import (
	"context"

	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// InstanceStore provisions and controls the power state of compute instances.
type InstanceStore interface {
	Create(ctx context.Context, domain *instancedom.Instance) error
	Delete(ctx context.Context, domain *instancedom.Instance) error
	PowerOn(ctx context.Context, domain *instancedom.Instance) error
	PowerOff(ctx context.Context, domain *instancedom.Instance) error
}
