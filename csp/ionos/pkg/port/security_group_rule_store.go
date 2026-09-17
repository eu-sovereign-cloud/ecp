package port

import (
	"context"

	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

// SecurityGroupRuleStore reconciles a standalone SECA security group rule. IONOS has no
// standalone rule object — a rule belongs to an NSG — so the rule is a reusable template that
// materialises through whichever security group pulls it in via ruleRefs.
type SecurityGroupRuleStore interface {
	Create(ctx context.Context, domain *securitygroupruledom.SecurityGroupRule) error
	Delete(ctx context.Context, domain *securitygroupruledom.SecurityGroupRule) error
}
