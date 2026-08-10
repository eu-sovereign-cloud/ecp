package kubernetes

import (
	"context"

	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

// SecurityGroupRulePlugin is implemented by CSP plugins that manage SecurityGroupRule resources.
type SecurityGroupRulePlugin interface {
	Create(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule) error
	Delete(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule) error
}
