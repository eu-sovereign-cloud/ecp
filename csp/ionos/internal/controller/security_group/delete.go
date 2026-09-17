package securitygroup

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

type DeleteSecurityGroup struct {
	Store port.SecurityGroupStore
}

func (c *DeleteSecurityGroup) Do(ctx context.Context, domain *securitygroupdom.SecurityGroup) error {
	return c.Store.Delete(ctx, domain)
}
