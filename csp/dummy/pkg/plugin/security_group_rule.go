package plugin

import (
	"context"
	"log/slog"
	"time"

	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

type SecurityGroupRule struct {
	logger *slog.Logger
}

func NewSecurityGroupRule(logger *slog.Logger) *SecurityGroupRule {
	return &SecurityGroupRule{logger: logger}
}

func (sgr *SecurityGroupRule) Create(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule) error {
	return simulateSecurityGroupRule(ctx, "create", resource, securityGroupRuleDelay(), sgr.logger)
}

func (sgr *SecurityGroupRule) Delete(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule) error {
	return simulateSecurityGroupRule(ctx, "delete", resource, securityGroupRuleDelay(), sgr.logger)
}

func securityGroupRuleDelay() time.Duration {
	return networkDelay()
}
