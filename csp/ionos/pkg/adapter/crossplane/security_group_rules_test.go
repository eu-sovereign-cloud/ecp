package crossplane

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

func sourceRefs(cidrs ...string) []commondomain.Reference {
	refs := make([]commondomain.Reference, 0, len(cidrs))
	for _, c := range cidrs {
		refs = append(refs, commondomain.Reference{Resource: c})
	}
	return refs
}

// A SECA rule's `from`/`to` range and its `list` are additive, not alternatives: the schema
// documents `list` as defining *additional* individual ports on top of the range. Treating them
// as alternatives would silently drop whichever the plugin did not pick.
func TestExpandPortsCombinesRangeAndList(t *testing.T) {
	tests := []struct {
		name string
		from int
		to   int
		list []int
		want []portRange
	}{
		{name: "no ports means every port", want: nil},
		{name: "from only is a single port", from: 22, want: []portRange{{From: 22, To: 22}}},
		{name: "to only is a single port", to: 443, want: []portRange{{From: 443, To: 443}}},
		{name: "from and to span a range", from: 8000, to: 8080, want: []portRange{{From: 8000, To: 8080}}},
		{name: "list yields one window per port", list: []int{80, 443}, want: []portRange{{From: 80, To: 80}, {From: 443, To: 443}}},
		{
			name: "range and list are additive",
			from: 8000, to: 8080, list: []int{22},
			want: []portRange{{From: 8000, To: 8080}, {From: 22, To: 22}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, expandPorts(tc.from, tc.to, tc.list))
		})
	}
}

func TestExpandProtocols(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		hasIcmp  bool
		want     []string
	}{
		{name: "tcp", protocol: "tcp", want: []string{protocolTCP}},
		{name: "udp", protocol: "udp", want: []string{protocolUDP}},
		{name: "icmp", protocol: "icmp", want: []string{protocolICMP}},
		{name: "tcp+udp has no single IONOS equivalent", protocol: "tcp+udp", want: []string{protocolTCP, protocolUDP}},
		{name: "empty means any", protocol: "", want: []string{protocolAny}},
		{name: "empty with an icmp block means icmp", protocol: "", hasIcmp: true, want: []string{protocolICMP}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := expandProtocols(tc.protocol, tc.hasIcmp)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	t.Run("an unmapped protocol is refused, not passed through", func(t *testing.T) {
		_, err := expandProtocols("sctp", false)
		require.ErrorIs(t, err, backend.ErrNotSupported)
	})
}

func TestFirewallType(t *testing.T) {
	require.Equal(t, firewallTypeEgress, firewallType("egress"))
	require.Equal(t, firewallTypeIngress, firewallType("ingress"))
	// SECA and IONOS both default an unset direction to ingress.
	require.Equal(t, firewallTypeIngress, firewallType(""))
}

// An IONOS NSG rule matches only on a source address. A sourceRef naming another SECA resource
// has no equivalent, and mapping it best-effort onto an IP would silently widen or narrow the
// rule, so it is refused outright.
func TestExpandSourcesRejectsNonAddressReferences(t *testing.T) {
	for _, res := range []string{
		"security-groups/web",
		"tenants/t1/workspaces/w1/security-groups/web",
		"instances/db-1",
		"internet-gateways/egress",
	} {
		t.Run(res, func(t *testing.T) {
			_, err := expandSources(normalizedRule{Origin: "web", SourceRef: sourceRefs(res)})
			require.ErrorIs(t, err, backend.ErrNotSupported)
		})
	}
}

func TestExpandSourcesAcceptsAddressesAndCIDRs(t *testing.T) {
	got, err := expandSources(normalizedRule{
		Origin:    "web",
		SourceRef: sourceRefs("10.0.0.0/8", "192.168.1.5", "2001:db8::/32"),
	})
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, "10.0.0.0/8", *got[0])
	require.Equal(t, "192.168.1.5", *got[1])
	require.Equal(t, "2001:db8::/32", *got[2])
}

// No sourceRef means "all traffic", which IONOS expresses as an unset sourceIp. It must still
// produce exactly one rule rather than none.
func TestExpandSourcesEmptyMeansOneUnsetSource(t *testing.T) {
	got, err := expandSources(normalizedRule{Origin: "web"})
	require.NoError(t, err)
	require.Equal(t, []*string{nil}, got)
}

// The Crossplane NSGFirewallRule exposes no ipVersion field, so a rule's family can only be
// implied by its source address. A version with no source cannot be honoured, and IONOS would
// silently default the rule to IPv4.
func TestExpandSourcesRejectsVersionWithoutSource(t *testing.T) {
	_, err := expandSources(normalizedRule{Origin: "web", Version: commondomain.IPVersionIPv6})
	require.ErrorIs(t, err, backend.ErrNotSupported)
}

// A version that contradicts the source family would apply a rule for the wrong family without
// any signal, since the version itself is never sent.
func TestExpandSourcesRejectsVersionContradictingSource(t *testing.T) {
	_, err := expandSources(normalizedRule{
		Origin:    "web",
		Version:   commondomain.IPVersionIPv6,
		SourceRef: sourceRefs("10.0.0.0/8"),
	})
	require.ErrorIs(t, err, backend.ErrNotSupported)

	// The agreeing case is accepted.
	_, err = expandSources(normalizedRule{
		Origin:    "web",
		Version:   commondomain.IPVersionIPv4,
		SourceRef: sourceRefs("10.0.0.0/8"),
	})
	require.NoError(t, err)
}

// One SECA rule fans out across (protocol x port x source): an IONOS rule carries exactly one
// of each.
func TestBuildRuleCRsFansOut(t *testing.T) {
	rules := normalizeInlineRules([]securitygroupdom.SecurityGroupRuleSpec{{
		Direction: "ingress",
		Protocol:  "tcp+udp",
		Ports:     &securitygroupdom.Ports{List: []int{80, 443}},
		SourceRef: sourceRefs("10.0.0.0/8", "192.168.0.0/16"),
	}}, "web")

	crs, err := buildRuleCRs("web", "workspace-1", "ns", "dc-ns", rules)
	require.NoError(t, err)
	// 2 protocols x 2 ports x 2 sources.
	require.Len(t, crs, 8)

	// Every variant gets its own CR, under its own content-derived name.
	names := map[string]bool{}
	for _, cr := range crs {
		require.True(t, strings.HasPrefix(cr.Name, "web-"), "name %q must be scoped to the group", cr.Name)
		names[cr.Name] = true
	}
	require.Len(t, names, 8, "distinct rules must not collide onto one name")

	for _, cr := range crs {
		require.Equal(t, "web", cr.Labels[LabelSecurityGroup])
		require.Equal(t, "ns", cr.Namespace)
		require.Equal(t, firewallTypeIngress, *cr.Spec.ForProvider.Type)
		require.Equal(t, "web", cr.Spec.ForProvider.NsgIDRef.Name)
		require.Equal(t, "ns", cr.Spec.ForProvider.NsgIDRef.Namespace)
		require.Equal(t, "workspace-1", cr.Spec.ForProvider.DatacenterIDRef.Name)
		require.Equal(t, "dc-ns", cr.Spec.ForProvider.DatacenterIDRef.Namespace)
		require.Equal(t, ProviderConfigName, cr.Spec.ProviderConfigReference.Name)
	}
}

// A rule with no ports still produces one rule — one matching every port — rather than none.
func TestBuildRuleCRsUnsetPortsMatchEveryPort(t *testing.T) {
	rules := normalizeInlineRules([]securitygroupdom.SecurityGroupRuleSpec{{
		Direction: "egress",
		Protocol:  "tcp",
	}}, "web")

	crs, err := buildRuleCRs("web", "workspace-1", "ns", "dc-ns", rules)
	require.NoError(t, err)
	require.Len(t, crs, 1)
	require.Nil(t, crs[0].Spec.ForProvider.PortRangeStart)
	require.Nil(t, crs[0].Spec.ForProvider.PortRangeEnd)
	require.Nil(t, crs[0].Spec.ForProvider.SourceIP)
	require.Equal(t, firewallTypeEgress, *crs[0].Spec.ForProvider.Type)
}

// IONOS accepts a port range only on TCP and UDP. Sending one on an ICMP or ANY rule would be
// rejected by the provider, so ports are dropped for those protocols.
func TestBuildRuleCRsOmitsPortsOnNonPortProtocols(t *testing.T) {
	rules := normalizeInlineRules([]securitygroupdom.SecurityGroupRuleSpec{{
		Direction: "ingress",
		Protocol:  "icmp",
		Ports:     &securitygroupdom.Ports{From: 80, To: 90},
		Icmp:      &securitygroupdom.IcmpConfig{Type: 8, Code: 0},
	}}, "web")

	crs, err := buildRuleCRs("web", "workspace-1", "ns", "dc-ns", rules)
	require.NoError(t, err)
	require.Len(t, crs, 1)
	require.Nil(t, crs[0].Spec.ForProvider.PortRangeStart)
	require.Equal(t, "8", *crs[0].Spec.ForProvider.IcmpType)
	require.Equal(t, "0", *crs[0].Spec.ForProvider.IcmpCode)
}

// An ICMP rule with no icmp block means "all types and codes", which IONOS expresses by leaving
// both unset. SECA's IcmpConfig has no unset marker — 0 is a valid type and a valid code — so
// the absent block is the only thing that can select the permissive default.
func TestBuildRuleCRsIcmpWithoutConfigLeavesTypeAndCodeUnset(t *testing.T) {
	rules := normalizeInlineRules([]securitygroupdom.SecurityGroupRuleSpec{{
		Direction: "ingress",
		Protocol:  "icmp",
	}}, "web")

	crs, err := buildRuleCRs("web", "workspace-1", "ns", "dc-ns", rules)
	require.NoError(t, err)
	require.Len(t, crs, 1)
	require.Nil(t, crs[0].Spec.ForProvider.IcmpType)
	require.Nil(t, crs[0].Spec.ForProvider.IcmpCode)
}

func TestBuildRuleCRsPropagatesRuleErrors(t *testing.T) {
	rules := normalizeInlineRules([]securitygroupdom.SecurityGroupRuleSpec{{
		Direction: "ingress",
		Protocol:  "tcp",
		SourceRef: sourceRefs("security-groups/other"),
	}}, "web")

	_, err := buildRuleCRs("web", "workspace-1", "ns", "dc-ns", rules)
	require.True(t, errors.Is(err, backend.ErrNotSupported))
}

// Rule CR names are derived from what a rule says, not where it sits in the spec. That is what
// lets an edit converge as a reap plus a create instead of an in-place rewrite the provider
// would reject (protocol is immutable) or fight (sourceIp/targetIp/type are late-initialised).
func TestRuleCRNameIsContentAddressed(t *testing.T) {
	ssh := []securitygroupdom.SecurityGroupRuleSpec{{
		Direction: "ingress", Protocol: "tcp", Ports: &securitygroupdom.Ports{From: 22},
	}}
	https := []securitygroupdom.SecurityGroupRuleSpec{{
		Direction: "ingress", Protocol: "tcp", Ports: &securitygroupdom.Ports{From: 443},
	}}

	build := func(specs []securitygroupdom.SecurityGroupRuleSpec) []string {
		crs, err := buildRuleCRs("web", "workspace-1", "ns", "dc-ns", normalizeInlineRules(specs, "web"))
		require.NoError(t, err)
		names := make([]string, 0, len(crs))
		for _, cr := range crs {
			names = append(names, cr.Name)
		}
		return names
	}

	require.Equal(t, build(ssh), build(ssh), "the same rule must always name the same CR")
	require.NotEqual(t, build(ssh), build(https), "a changed port must name a different CR")

	// Reordering a group's rules must not churn any CR.
	forward := build(append(append([]securitygroupdom.SecurityGroupRuleSpec{}, ssh...), https...))
	reversed := build(append(append([]securitygroupdom.SecurityGroupRuleSpec{}, https...), ssh...))
	require.ElementsMatch(t, forward, reversed)
}
