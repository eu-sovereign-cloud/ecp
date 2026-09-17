package securitygrouprule

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

type DeleteSecurityGroupRule struct {
	Store port.SecurityGroupRuleStore
}

func (c *DeleteSecurityGroupRule) Do(ctx context.Context, domain *securitygroupruledom.SecurityGroupRule) error {
	return c.Store.Delete(ctx, domain)
}
