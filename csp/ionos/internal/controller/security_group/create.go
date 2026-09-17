package securitygroup

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

type CreateSecurityGroup struct {
	Store port.SecurityGroupStore
}

func (c *CreateSecurityGroup) Do(ctx context.Context, domain *securitygroupdom.SecurityGroup) error {
	return c.Store.Create(ctx, domain)
}
