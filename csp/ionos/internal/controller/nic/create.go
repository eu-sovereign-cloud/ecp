package nic

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

type CreateNic struct {
	Store port.NicStore
}

func (c *CreateNic) Do(ctx context.Context, domain *nicdom.Nic) error {
	return c.Store.Create(ctx, domain)
}
