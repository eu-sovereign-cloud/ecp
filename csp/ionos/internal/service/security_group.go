package service

import (
	"context"

	securitygroupctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/security_group"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	securitygroupk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
)

var _ securitygroupk8s.SecurityGroupPlugin = (*SecurityGroup)(nil)

type SecurityGroup struct {
	Creator *securitygroupctrl.CreateSecurityGroup
	Deleter *securitygroupctrl.DeleteSecurityGroup
}

func (s *SecurityGroup) Create(ctx context.Context, resource *securitygroupdom.SecurityGroup) error {
	return s.Creator.Do(ctx, resource)
}

func (s *SecurityGroup) Delete(ctx context.Context, resource *securitygroupdom.SecurityGroup) error {
	return s.Deleter.Do(ctx, resource)
}

// Update re-runs the create path, which is a full reconcile of the group's rules against its
// current spec rather than a one-shot provisioning step: it reaps the rule CRs the spec no
// longer asks for and creates the ones it does. Over an unchanged spec that reads and writes
// nothing, which is what the level-triggered Update contract requires.
//
// This is deliberately not the blanket "do nothing" the other IONOS resources use (see
// update.go). Dropping an edit silently is a tolerable cost for a label or a size; for a
// security group it would mean a tenant revoking an allow-rule, being told the group is active,
// and having the traffic keep flowing. The NSG model is what makes the alternative cheap —
// rules are addressed by content, so the diff is a set comparison and no rule is ever rewritten
// in place.
func (s *SecurityGroup) Update(ctx context.Context, resource *securitygroupdom.SecurityGroup) error {
	return s.Creator.Do(ctx, resource)
}
