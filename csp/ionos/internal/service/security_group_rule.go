package service

import (
	"context"

	securitygrouprulectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/security_group_rule"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	securitygrouprulek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
)

var _ securitygrouprulek8s.SecurityGroupRulePlugin = (*SecurityGroupRule)(nil)

type SecurityGroupRule struct {
	Creator *securitygrouprulectrl.CreateSecurityGroupRule
	Deleter *securitygrouprulectrl.DeleteSecurityGroupRule
}

func (s *SecurityGroupRule) Create(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule) error {
	return s.Creator.Do(ctx, resource)
}

func (s *SecurityGroupRule) Delete(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule) error {
	return s.Deleter.Do(ctx, resource)
}

// Update does nothing. A standalone rule owns no IONOS resource of its own, so there is nothing
// here to reconcile; re-running Create would only re-announce the declaration on every pass.
//
// An edit to the rule does reach IONOS, just not through here: every security group that names
// it in ruleRefs re-resolves it on its own reconcile and rebuilds its NSGFirewallRule CRs from
// what the rule now says.
func (s *SecurityGroupRule) Update(_ context.Context, _ *securitygroupruledom.SecurityGroupRule) error {
	return nil
}
