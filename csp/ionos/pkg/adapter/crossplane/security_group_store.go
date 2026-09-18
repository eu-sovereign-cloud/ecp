package crossplane

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	sgrk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
)

var _ port.SecurityGroupStore = (*SecurityGroupStore)(nil)

// SecurityGroupStore maps a SECA security group onto an IONOS Network Security Group.
//
// An NSG is datacenter-scoped, named and reusable, which is exactly the shape of a SECA
// security group, so this is an ordinary eager resource: the NSG is provisioned on Create and
// removed on Delete, independently of any instance. The rules it carries are the same firewall
// rule objects the classic per-NIC firewall uses, only parented to the group instead of to one
// NIC — which is what lets the group exist before anything attaches it.
//
// Attaching the group to a NIC is the Instance plugin's job (the NIC's securityGroupsIdsRefs);
// nothing here touches a NIC.
type SecurityGroupStore struct {
	base
}

func NewSecurityGroupStore(c client.Client, logger *slog.Logger) *SecurityGroupStore {
	return &SecurityGroupStore{base{client: c, logger: logger}}
}

// securityGroupNamespace is where a security group's NSG and its rules live.
//
// Security group names are only workspace-unique, not tenant-unique, so the workspace has to be
// in the hash: two workspaces of one tenant can both hold a group named "web", and without it
// the second would adopt the first's NSG — in the wrong datacenter. It is also where the Nic
// CRs live, which keeps a NIC's securityGroupsIdsRefs a same-namespace reference.
func securityGroupNamespace(domain *securitygroupdom.SecurityGroup) string {
	return k8sadapter.ComputeNamespace(&kernelresource.Scope{
		Tenant:    domain.GetTenant(),
		Workspace: domain.GetWorkspace(),
	})
}

// securityGroupDatacenterNamespace is the tenant-wide namespace the workspace's Datacenter CR
// lives in, matching where the Workspace plugin writes it.
func securityGroupDatacenterNamespace(domain *securitygroupdom.SecurityGroup) string {
	return k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: domain.GetTenant()})
}

func (a *SecurityGroupStore) Create(ctx context.Context, domain *securitygroupdom.SecurityGroup) error {
	ns := securityGroupNamespace(domain)
	dcNs := securityGroupDatacenterNamespace(domain)

	// 1. An NSG belongs to a datacenter, so the workspace must be provisioned first.
	dc := &ionosv1alpha1.Datacenter{
		TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Datacenter_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetWorkspace(), Namespace: dcNs},
	}
	if err := a.checkExisting(ctx, dc); err != nil {
		return fmt.Errorf("security group %q requires workspace datacenter %q: %w",
			domain.GetName(), domain.GetWorkspace(), err)
	}

	// 2. The NSG itself. Every rule references it, so it has to be ready before they are
	// written — createCR returns StillProcessing until it is.
	if err := a.createCR(ctx, newNSG(domain, ns, dcNs)); err != nil {
		return err
	}

	// 3. The rules: the group's own inline rules plus every shared rule it pulls in.
	//
	// A ref that does not resolve yet holds the group open, but it must not hold back the rules
	// that did resolve. Create is also the update path (see service.SecurityGroup.Update), so the
	// same edit that names a not-yet-existing rule may have revoked another one; returning here
	// would leave that revoked rule in place and keep allowing traffic the tenant withdrew.
	// Reconciling the resolvable subset is safe because rule CR names are content-derived (see
	// ruleCRName): the subset produces exactly the names it would produce in the full set, so the
	// reap removes only what the spec dropped and nothing flaps when the missing ref arrives.
	rules, unresolved, err := a.resolveRules(ctx, domain)
	if err != nil {
		return err
	}
	desired, err := buildRuleCRs(domain.GetName(), domain.GetWorkspace(), ns, dcNs, rules)
	if err != nil {
		return err
	}
	if err := a.reconcileRules(ctx, domain.GetName(), ns, desired); err != nil {
		return err
	}
	if unresolved {
		return backend.StillProcessing
	}
	return nil
}

func (a *SecurityGroupStore) Delete(ctx context.Context, domain *securitygroupdom.SecurityGroup) error {
	ns := securityGroupNamespace(domain)

	// Rules first — they belong to the NSG. They are reaped by label rather than by rebuilding
	// the names the current spec would produce, so rules left behind by an earlier spec go too.
	// Reaping every rule of the group is the same operation as reconciling towards an empty
	// desired set.
	pending, err := a.reapRules(ctx, domain.GetName(), ns, nil)
	if err != nil {
		return err
	}
	if pending {
		return backend.StillProcessing
	}

	return a.deleteCR(ctx, &ionosv1alpha1.NSG{
		TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.NSG_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetName(), Namespace: ns},
	})
}

// reconcileRules drives the group's rule CRs towards desired: it removes the ones the spec no
// longer asks for and creates the ones it does.
//
// This is what makes the store level-triggered rather than create-only. Rule CR names are
// content-derived (see ruleCRName), so an edited rule arrives here as one name disappearing and
// another appearing — the edit converges through a delete and a create, and no rule is ever
// rewritten in place. In a steady state nothing matches the reap and every create comes back
// AlreadyExists, so a reconcile that changes nothing writes nothing.
//
// Both halves run in a single pass rather than returning on the first CR still converging: a
// group of N rules would otherwise take N requeues to come up. The intermediate states are safe
// — NSG rules only ever allow traffic, so a group missing some of them is more restrictive than
// asked for, never less.
func (a *SecurityGroupStore) reconcileRules(ctx context.Context, sgName, ns string, desired []*ionosv1alpha1.NSGFirewallRule) error {
	want := make(map[string]struct{}, len(desired))
	for _, cr := range desired {
		want[cr.Name] = struct{}{}
	}

	pending, err := a.reapRules(ctx, sgName, ns, want)
	if err != nil {
		return err
	}

	for _, cr := range desired {
		switch err := a.createCR(ctx, cr); {
		case err == nil:
		case errors.Is(err, backend.StillProcessing):
			pending = true
		default:
			return err
		}
	}

	if pending {
		return backend.StillProcessing
	}
	return nil
}

// reapRules deletes every rule CR of the group that keep does not name, reporting whether any
// deletion is still in flight.
//
// The list is bound to the group's own namespace as well as the label: a group name is only
// workspace-unique, so the label alone would also match a sibling workspace's group of the same
// name and strip the rules off a live NSG there.
func (a *SecurityGroupStore) reapRules(ctx context.Context, sgName, ns string, keep map[string]struct{}) (pending bool, err error) {
	var rules ionosv1alpha1.NSGFirewallRuleList
	if err := a.client.List(ctx, &rules,
		client.InNamespace(ns),
		client.MatchingLabels{LabelSecurityGroup: sgName},
	); err != nil {
		return false, fmt.Errorf("list rules of security group %q: %w", sgName, err)
	}

	for i := range rules.Items {
		rule := &rules.Items[i]
		if _, ok := keep[rule.Name]; ok {
			continue
		}
		rule.TypeMeta = metav1.TypeMeta{Kind: ionosv1alpha1.NSGFirewallRule_Kind}
		switch err := a.deleteCR(ctx, rule); {
		case err == nil:
		case errors.Is(err, backend.StillProcessing):
			pending = true
		default:
			return false, err
		}
	}
	return pending, nil
}

// resolveRules collects the group's inline rules and every standalone SecurityGroupRule it
// references, normalised to one shape.
//
// A ref naming a rule that does not exist yet is not an error: the group is level-triggered, so
// the rule may still arrive. It is reported through the second return value instead of aborting
// the whole resolution, which lets the caller apply the rules that did resolve and requeue for
// the rest — see Create.
func (a *SecurityGroupStore) resolveRules(ctx context.Context, domain *securitygroupdom.SecurityGroup) ([]normalizedRule, bool, error) {
	rules := normalizeInlineRules(domain.Spec.Rules, domain.GetName())
	unresolved := false

	for _, ref := range domain.Spec.RuleRefs {
		target := commonbackend.ParseReference(ref, domain.GetTenant())
		if target.Workspace == "" {
			target.Workspace = domain.GetWorkspace()
		}
		ns := k8sadapter.ComputeNamespace(&kernelresource.Scope{
			Tenant:    target.Tenant,
			Workspace: target.Workspace,
		})

		cr := &sgrk8s.SecurityGroupRule{}
		if err := a.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: target.Name}, cr); err != nil {
			if apierrors.IsNotFound(err) {
				// A shared rule the group names but that does not exist yet. It may still
				// arrive — the group is level-triggered — so keep going and let the caller
				// requeue rather than failing or dropping the rules that did resolve.
				a.logger.Info("security group: waiting for referenced rule",
					"security_group", domain.GetName(), "rule", target.Name)
				unresolved = true
				continue
			}
			return nil, false, fmt.Errorf("read security group rule %q: %w", target.Name, err)
		}

		rule, err := sgrk8s.SecurityGroupRuleFromCR(cr)
		if err != nil {
			return nil, false, fmt.Errorf("convert security group rule %q: %w", target.Name, err)
		}
		rules = append(rules, normalizeStandaloneRule(rule.Spec, target.Name))
	}

	return rules, unresolved, nil
}

// newNSG builds the NSG backing a SECA security group. IONOS requires a description and SECA
// models none, so one is derived from the group's own identity — deterministically, so it never
// reads as drift on a later reconcile.
func newNSG(domain *securitygroupdom.SecurityGroup, namespace, datacenterNamespace string) *ionosv1alpha1.NSG {
	return &ionosv1alpha1.NSG{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ionosv1alpha1.CRDGroupVersion.String(),
			Kind:       ionosv1alpha1.NSG_Kind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetName(), Namespace: namespace},
		Spec: ionosv1alpha1.NSGSpec{
			ForProvider: ionosv1alpha1.NSGParameters{
				Name:        new(domain.GetName()),
				Description: new(fmt.Sprintf("SECA security group %s/%s", domain.GetWorkspace(), domain.GetName())),
				DatacenterIDRef: &v1.NamespacedReference{
					Name:      domain.GetWorkspace(),
					Namespace: datacenterNamespace,
				},
			},
			ManagedResourceSpec: providerConfig(),
		},
	}
}
