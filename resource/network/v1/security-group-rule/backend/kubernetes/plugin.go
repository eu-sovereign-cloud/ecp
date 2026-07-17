package kubernetes

import (
	"context"

	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

// SecurityGroupRulePlugin is implemented by CSP plugins that manage SecurityGroupRule resources.
type SecurityGroupRulePlugin interface {
	Create(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule) error
	Delete(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule) error
}
