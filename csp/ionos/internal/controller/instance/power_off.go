package instance

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

type PowerOffInstance struct {
	Store port.InstanceStore
}

func (c *PowerOffInstance) Do(ctx context.Context, domain *instancedom.Instance) error {
	return c.Store.PowerOff(ctx, domain)
}
