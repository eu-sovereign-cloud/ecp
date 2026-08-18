package instance

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

type DeleteInstance struct {
	Store port.InstanceStore
}

func (c *DeleteInstance) Do(ctx context.Context, domain *instancedom.Instance) error {
	return c.Store.Delete(ctx, domain)
}
