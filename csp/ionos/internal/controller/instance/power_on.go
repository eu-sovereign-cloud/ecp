package instance

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

type PowerOnInstance struct {
	Store port.InstanceStore
}

func (c *PowerOnInstance) Do(ctx context.Context, domain *instancedom.Instance) error {
	return c.Store.PowerOn(ctx, domain)
}
