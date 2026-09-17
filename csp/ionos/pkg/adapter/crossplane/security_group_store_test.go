package crossplane

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/require"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	sgrk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
)

func securityGroupScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, ionosv1alpha1.AddToScheme(s))
	require.NoError(t, sgrk8s.AddToScheme(s))
	return s
}

func testSecurityGroup() *securitygroupdom.SecurityGroup {
	sg := &securitygroupdom.SecurityGroup{}
	sg.Name = "web"
	sg.Scope = resource.Scope{Tenant: "tenant-1", Workspace: "workspace-1"}
	sg.Region = "regionBerlin"
	return sg
}

// readySecurityGroupDatacenter builds the workspace's Datacenter in the tenant-wide namespace
// the Workspace plugin writes it to, already Ready.
func readySecurityGroupDatacenter() *ionosv1alpha1.Datacenter {
	dc := &ionosv1alpha1.Datacenter{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "workspace-1",
			Namespace:  k8sadapter.ComputeNamespace(&resource.Scope{Tenant: "tenant-1"}),
			Generation: 1,
		},
	}
	dc.SetConditions(v1.Available().WithObservedGeneration(1))
	return dc
}

func readyNSG(ns string) *ionosv1alpha1.NSG {
	nsg := &ionosv1alpha1.NSG{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: ns, Generation: 1},
	}
	nsg.SetConditions(v1.Available().WithObservedGeneration(1))
	return nsg
}

// An NSG belongs to a datacenter, so a security group cannot be provisioned before its
// workspace is. Creating it anyway would hand IONOS a dangling datacenter reference.
func TestSecurityGroupCreateRequiresWorkspaceDatacenter(t *testing.T) {
	c := fakeclient.NewClientBuilder().WithScheme(securityGroupScheme(t)).Build()
	store := NewSecurityGroupStore(c, testLogger())

	err := store.Create(context.Background(), testSecurityGroup())
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires workspace datacenter")
}

// The first reconcile creates the NSG and waits: its rules reference it, so none may be written
// until the provider reports it ready.
func TestSecurityGroupCreateCreatesNSGAndWaits(t *testing.T) {
	sg := testSecurityGroup()
	sg.Spec.Rules = []securitygroupdom.SecurityGroupRuleSpec{{
		Direction: "ingress", Protocol: "tcp", Ports: &securitygroupdom.Ports{From: 22},
	}}
	ns := securityGroupNamespace(sg)

	c := fakeclient.NewClientBuilder().
		WithScheme(securityGroupScheme(t)).
		WithObjects(readySecurityGroupDatacenter()).
		Build()
	store := NewSecurityGroupStore(c, testLogger())

	err := store.Create(context.Background(), sg)
	require.ErrorIs(t, err, backend.StillProcessing)

	nsg := &ionosv1alpha1.NSG{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "web"}, nsg))
	require.Equal(t, "web", *nsg.Spec.ForProvider.Name)
	require.NotEmpty(t, *nsg.Spec.ForProvider.Description, "IONOS requires a description")
	require.Equal(t, "workspace-1", nsg.Spec.ForProvider.DatacenterIDRef.Name)

	// No rule may exist yet.
	var rules ionosv1alpha1.NSGFirewallRuleList
	require.NoError(t, c.List(context.Background(), &rules))
	require.Empty(t, rules.Items)
}

// Once the NSG is ready every rule is issued in the same pass. Returning on the first one still
// converging would provision one rule per reconcile.
func TestSecurityGroupCreateWritesAllRulesInOnePass(t *testing.T) {
	sg := testSecurityGroup()
	sg.Spec.Rules = []securitygroupdom.SecurityGroupRuleSpec{
		{Direction: "ingress", Protocol: "tcp", Ports: &securitygroupdom.Ports{From: 22}},
		{Direction: "ingress", Protocol: "tcp", Ports: &securitygroupdom.Ports{List: []int{80, 443}}},
		{Direction: "egress", Protocol: ""},
	}
	ns := securityGroupNamespace(sg)

	c := fakeclient.NewClientBuilder().
		WithScheme(securityGroupScheme(t)).
		WithObjects(readySecurityGroupDatacenter(), readyNSG(ns)).
		Build()
	store := NewSecurityGroupStore(c, testLogger())

	require.ErrorIs(t, store.Create(context.Background(), sg), backend.StillProcessing)

	var rules ionosv1alpha1.NSGFirewallRuleList
	require.NoError(t, c.List(context.Background(), &rules, client.InNamespace(ns)))
	// 1 (ssh) + 2 (port list) + 1 (egress any) = 4.
	require.Len(t, rules.Items, 4)
	for _, r := range rules.Items {
		require.Equal(t, "web", r.Labels[LabelSecurityGroup])
	}
}

// Re-running Create against rules that already exist must converge rather than fail on
// AlreadyExists — every store method runs under at-least-once reconciliation.
func TestSecurityGroupCreateIsIdempotent(t *testing.T) {
	sg := testSecurityGroup()
	sg.Spec.Rules = []securitygroupdom.SecurityGroupRuleSpec{{
		Direction: "ingress", Protocol: "tcp", Ports: &securitygroupdom.Ports{From: 22},
	}}
	ns := securityGroupNamespace(sg)

	c := fakeclient.NewClientBuilder().
		WithScheme(securityGroupScheme(t)).
		WithObjects(readySecurityGroupDatacenter(), readyNSG(ns)).
		Build()
	store := NewSecurityGroupStore(c, testLogger())

	require.ErrorIs(t, store.Create(context.Background(), sg), backend.StillProcessing)
	// Second pass: the rule exists but is not ready yet, so still converging — not an error.
	require.ErrorIs(t, store.Create(context.Background(), sg), backend.StillProcessing)

	var rules ionosv1alpha1.NSGFirewallRuleList
	require.NoError(t, c.List(context.Background(), &rules, client.InNamespace(ns)))
	require.Len(t, rules.Items, 1, "the second pass must not duplicate the rule")
}

// A group that names a shared rule which does not exist yet waits for it. The group is
// level-triggered, so the rule may still arrive; failing would strand the group in error.
func TestSecurityGroupCreateWaitsForMissingRuleRef(t *testing.T) {
	sg := testSecurityGroup()
	sg.Spec.RuleRefs = []commondomain.Reference{{Resource: "security-group-rules/shared-ssh"}}
	ns := securityGroupNamespace(sg)

	c := fakeclient.NewClientBuilder().
		WithScheme(securityGroupScheme(t)).
		WithObjects(readySecurityGroupDatacenter(), readyNSG(ns)).
		Build()
	store := NewSecurityGroupStore(c, testLogger())

	require.ErrorIs(t, store.Create(context.Background(), sg), backend.StillProcessing)

	var rules ionosv1alpha1.NSGFirewallRuleList
	require.NoError(t, c.List(context.Background(), &rules, client.InNamespace(ns)))
	require.Empty(t, rules.Items, "no rule may be written while a referenced rule is unresolved")
}

// A referenced shared rule is materialised as part of the group that pulls it in, alongside the
// group's own inline rules.
func TestSecurityGroupCreateMaterialisesRuleRefs(t *testing.T) {
	sg := testSecurityGroup()
	sg.Spec.Rules = []securitygroupdom.SecurityGroupRuleSpec{{
		Direction: "ingress", Protocol: "tcp", Ports: &securitygroupdom.Ports{From: 22},
	}}
	sg.Spec.RuleRefs = []commondomain.Reference{{Resource: "security-group-rules/shared-https"}}
	ns := securityGroupNamespace(sg)

	shared := &sgrk8s.SecurityGroupRule{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-https", Namespace: ns},
		Spec: sgrk8s.SecurityGroupRuleSpec{
			Direction: "ingress",
			Protocol:  "tcp",
			Ports:     &sgrk8s.Ports{From: 443},
		},
	}

	c := fakeclient.NewClientBuilder().
		WithScheme(securityGroupScheme(t)).
		WithObjects(readySecurityGroupDatacenter(), readyNSG(ns), shared).
		Build()
	store := NewSecurityGroupStore(c, testLogger())

	require.ErrorIs(t, store.Create(context.Background(), sg), backend.StillProcessing)

	var rules ionosv1alpha1.NSGFirewallRuleList
	require.NoError(t, c.List(context.Background(), &rules, client.InNamespace(ns)))
	require.Len(t, rules.Items, 2)

	ports := map[float64]bool{}
	for _, r := range rules.Items {
		require.NotNil(t, r.Spec.ForProvider.PortRangeStart)
		ports[*r.Spec.ForProvider.PortRangeStart] = true
	}
	require.True(t, ports[22], "the inline rule must be present")
	require.True(t, ports[443], "the referenced shared rule must be present")
}

// A rule the plugin cannot express must fail terminally rather than be silently dropped, which
// would report a security guarantee the group does not actually carry.
func TestSecurityGroupCreateRefusesUnexpressibleRule(t *testing.T) {
	sg := testSecurityGroup()
	sg.Spec.Rules = []securitygroupdom.SecurityGroupRuleSpec{{
		Direction: "ingress",
		Protocol:  "tcp",
		SourceRef: []commondomain.Reference{{Resource: "security-groups/other"}},
	}}
	ns := securityGroupNamespace(sg)

	c := fakeclient.NewClientBuilder().
		WithScheme(securityGroupScheme(t)).
		WithObjects(readySecurityGroupDatacenter(), readyNSG(ns)).
		Build()
	store := NewSecurityGroupStore(c, testLogger())

	err := store.Create(context.Background(), sg)
	require.True(t, errors.Is(err, backend.ErrNotSupported))
}

// Delete reaps rules by label rather than by rebuilding the names the current spec would
// produce, so a rule left behind by an earlier spec goes too.
func TestSecurityGroupDeleteReapsRulesThenNSG(t *testing.T) {
	sg := testSecurityGroup()
	ns := securityGroupNamespace(sg)

	stale := &ionosv1alpha1.NSGFirewallRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-r7",
			Namespace: ns,
			Labels:    map[string]string{LabelSecurityGroup: "web"},
		},
	}
	c := fakeclient.NewClientBuilder().
		WithScheme(securityGroupScheme(t)).
		WithObjects(readyNSG(ns), stale).
		Build()
	store := NewSecurityGroupStore(c, testLogger())

	require.NoError(t, store.Delete(context.Background(), sg))

	var rules ionosv1alpha1.NSGFirewallRuleList
	require.NoError(t, c.List(context.Background(), &rules))
	require.Empty(t, rules.Items)

	nsg := &ionosv1alpha1.NSG{}
	err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "web"}, nsg)
	require.Error(t, err, "the NSG must be deleted once its rules are gone")
}

// A group name is only workspace-unique, so the reap is bound to the group's own namespace as
// well as the label — otherwise deleting one workspace's group would strip the rules off a
// sibling workspace's live NSG of the same name.
func TestSecurityGroupDeleteIsScopedToItsWorkspace(t *testing.T) {
	sg := testSecurityGroup()
	ns := securityGroupNamespace(sg)

	sibling := testSecurityGroup()
	sibling.Scope = resource.Scope{Tenant: "tenant-1", Workspace: "workspace-2"}
	siblingNs := securityGroupNamespace(sibling)
	require.NotEqual(t, ns, siblingNs)

	siblingRule := &ionosv1alpha1.NSGFirewallRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-r0",
			Namespace: siblingNs,
			Labels:    map[string]string{LabelSecurityGroup: "web"},
		},
	}
	c := fakeclient.NewClientBuilder().
		WithScheme(securityGroupScheme(t)).
		WithObjects(readyNSG(ns), siblingRule).
		Build()
	store := NewSecurityGroupStore(c, testLogger())

	require.NoError(t, store.Delete(context.Background(), sg))

	var rules ionosv1alpha1.NSGFirewallRuleList
	require.NoError(t, c.List(context.Background(), &rules, client.InNamespace(siblingNs)))
	require.Len(t, rules.Items, 1, "the sibling workspace's rule must survive")
}

// ruleNames lists the rule CRs currently present for a group.
func ruleNames(t *testing.T, c client.Client, ns string) []string {
	t.Helper()
	var rules ionosv1alpha1.NSGFirewallRuleList
	require.NoError(t, c.List(context.Background(), &rules, client.InNamespace(ns)))
	names := make([]string, 0, len(rules.Items))
	for _, r := range rules.Items {
		names = append(names, r.Name)
	}
	return names
}

// markRulesReady flips every rule CR of a group to Ready so the next reconcile sees a settled
// steady state rather than one still converging.
func markRulesReady(t *testing.T, c client.Client, ns string) {
	t.Helper()
	var rules ionosv1alpha1.NSGFirewallRuleList
	require.NoError(t, c.List(context.Background(), &rules, client.InNamespace(ns)))
	for i := range rules.Items {
		r := &rules.Items[i]
		r.Generation = 1
		r.SetConditions(v1.Available().WithObservedGeneration(1))
		require.NoError(t, c.Update(context.Background(), r))
	}
}

// A rule dropped from the spec must be removed from IONOS. Leaving it would keep allowing
// traffic the tenant has revoked — the failure that matters most here.
func TestSecurityGroupCreateReapsRulesTheSpecDropped(t *testing.T) {
	sg := testSecurityGroup()
	sg.Spec.Rules = []securitygroupdom.SecurityGroupRuleSpec{
		{Direction: "ingress", Protocol: "tcp", Ports: &securitygroupdom.Ports{From: 22}},
		{Direction: "ingress", Protocol: "tcp", Ports: &securitygroupdom.Ports{From: 443}},
	}
	ns := securityGroupNamespace(sg)

	c := fakeclient.NewClientBuilder().
		WithScheme(securityGroupScheme(t)).
		WithObjects(readySecurityGroupDatacenter(), readyNSG(ns)).
		Build()
	store := NewSecurityGroupStore(c, testLogger())

	require.ErrorIs(t, store.Create(context.Background(), sg), backend.StillProcessing)
	require.Len(t, ruleNames(t, c, ns), 2)

	// The tenant revokes the HTTPS rule.
	sg.Spec.Rules = sg.Spec.Rules[:1]
	require.ErrorIs(t, store.Create(context.Background(), sg), backend.StillProcessing)
	require.Len(t, ruleNames(t, c, ns), 1, "the revoked rule must be gone")
}

// An edited rule must converge. Because names are content-derived the edit arrives as one CR
// disappearing and another appearing, so no rule is ever rewritten in place.
func TestSecurityGroupCreateConvergesAnEditedRule(t *testing.T) {
	sg := testSecurityGroup()
	sg.Spec.Rules = []securitygroupdom.SecurityGroupRuleSpec{{
		Direction: "ingress", Protocol: "tcp", Ports: &securitygroupdom.Ports{From: 22},
	}}
	ns := securityGroupNamespace(sg)

	c := fakeclient.NewClientBuilder().
		WithScheme(securityGroupScheme(t)).
		WithObjects(readySecurityGroupDatacenter(), readyNSG(ns)).
		Build()
	store := NewSecurityGroupStore(c, testLogger())

	require.ErrorIs(t, store.Create(context.Background(), sg), backend.StillProcessing)
	before := ruleNames(t, c, ns)
	require.Len(t, before, 1)

	// Move SSH off the default port.
	sg.Spec.Rules[0].Ports = &securitygroupdom.Ports{From: 2222}
	require.ErrorIs(t, store.Create(context.Background(), sg), backend.StillProcessing)

	after := ruleNames(t, c, ns)
	require.Len(t, after, 1, "the old rule must not linger alongside the new one")
	require.NotEqual(t, before[0], after[0])

	var rules ionosv1alpha1.NSGFirewallRuleList
	require.NoError(t, c.List(context.Background(), &rules, client.InNamespace(ns)))
	require.Equal(t, float64(2222), *rules.Items[0].Spec.ForProvider.PortRangeStart)
}

// Create is level-triggered: it runs on every reconcile of an active group, so a pass over a
// spec that has not changed must write nothing. An unconditional write would fight the
// provider's own reconcile forever.
func TestSecurityGroupCreateWritesNothingWhenNothingDrifted(t *testing.T) {
	sg := testSecurityGroup()
	sg.Spec.Rules = []securitygroupdom.SecurityGroupRuleSpec{
		{Direction: "ingress", Protocol: "tcp", Ports: &securitygroupdom.Ports{From: 22}},
		{Direction: "egress", Protocol: ""},
	}
	ns := securityGroupNamespace(sg)

	c := fakeclient.NewClientBuilder().
		WithScheme(securityGroupScheme(t)).
		WithObjects(readySecurityGroupDatacenter(), readyNSG(ns)).
		Build()
	store := NewSecurityGroupStore(c, testLogger())

	require.ErrorIs(t, store.Create(context.Background(), sg), backend.StillProcessing)
	markRulesReady(t, c, ns)

	var settled ionosv1alpha1.NSGFirewallRuleList
	require.NoError(t, c.List(context.Background(), &settled, client.InNamespace(ns)))
	versions := map[string]string{}
	for _, r := range settled.Items {
		versions[r.Name] = r.ResourceVersion
	}

	// Everything the spec asks for exists and is ready, so the group is done.
	require.NoError(t, store.Create(context.Background(), sg))

	var after ionosv1alpha1.NSGFirewallRuleList
	require.NoError(t, c.List(context.Background(), &after, client.InNamespace(ns)))
	require.Len(t, after.Items, len(versions))
	for _, r := range after.Items {
		require.Equal(t, versions[r.Name], r.ResourceVersion, "rule %q was rewritten by a no-op reconcile", r.Name)
	}
}
