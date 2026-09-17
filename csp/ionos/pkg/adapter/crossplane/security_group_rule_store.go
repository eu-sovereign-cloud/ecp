package crossplane

import (
	"context"
	"log/slog"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

var _ port.SecurityGroupRuleStore = (*SecurityGroupRuleStore)(nil)

type SecurityGroupRuleStore struct {
	base
}

func NewSecurityGroupRuleStore(c client.Client, logger *slog.Logger) *SecurityGroupRuleStore {
	return &SecurityGroupRuleStore{base{client: c, logger: logger}}
}

// Create marks the rule ready. IONOS has no standalone firewall rule: a rule belongs to an NSG
// (or, in the classic model, to a NIC), so a SECA SecurityGroupRule on its own has nothing to
// provision. It is a reusable template that takes effect through every security group naming it
// in ruleRefs, and the SecurityGroupStore writes it as one of that group's NSGFirewallRule CRs.
func (a *SecurityGroupRuleStore) Create(ctx context.Context, domain *securitygroupruledom.SecurityGroupRule) error {
	a.logger.Info("security group rule: ready as declaration, materialised by the groups that reference it",
		"security_group_rule", domain.GetName())
	return nil
}

// Delete is a no-op: the rule owns no IONOS resource. The NSGFirewallRule CRs built from it
// belong to the security groups that reference it and are reaped when those groups are deleted
// — a rule that is still referenced must not strip itself out of a live group.
func (a *SecurityGroupRuleStore) Delete(ctx context.Context, domain *securitygroupruledom.SecurityGroupRule) error {
	return nil
}
