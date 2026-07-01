package nic

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

type DeleteNic struct {
	Store port.NicStore
}

func (d *DeleteNic) Do(ctx context.Context, domain *nicdom.Nic) error {
	return d.Store.Delete(ctx, domain)
}
