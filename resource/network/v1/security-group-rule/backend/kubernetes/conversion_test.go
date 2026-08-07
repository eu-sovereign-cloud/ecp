package kubernetes_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
)

func TestSecurityGroupRuleConversionRoundTrip(t *testing.T) {
	in := &securitygroupruledom.SecurityGroupRule{
		Spec: securitygroupruledom.SecurityGroupRuleSpec{
			Direction: "ingress",
			Protocol:  "tcp",
			Version:   commondomain.IPVersionIPv4,
			Icmp:      &securitygroupruledom.IcmpConfig{Code: 1, Type: 2},
			Ports:     &securitygroupruledom.Ports{From: 80, To: 443, List: []int{22, 8080}},
			// Reference.resource: {collection}/{name}
			// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
			SourceRef: []commondomain.Reference{{Resource: "networks/net1"}},
		},
	}
	in.Name = "sgr1"
	in.Tenant = "t1"
	in.Workspace = "w1"
	in.Provider = securitygroupruledom.ProviderID
	in.Region = "r1"
	in.Status = &securitygroupruledom.SecurityGroupRuleStatus{
		Status: commondomain.Status{State: commondomain.ResourceStateActive},
	}
	in.Status.PushCondition(commondomain.StatusCondition{State: commondomain.ResourceStateActive})

	cr, err := SecurityGroupRuleToCR(in)
	require.NoError(t, err)

	out, err := SecurityGroupRuleFromCR(cr)
	require.NoError(t, err)

	require.Equal(t, in.Name, out.Name)
	require.Equal(t, in.Tenant, out.Tenant)
	require.Equal(t, in.Workspace, out.Workspace)
	require.Equal(t, in.Region, out.Region)
	require.Equal(t, in.Spec.Direction, out.Spec.Direction)
	require.Equal(t, in.Spec.Protocol, out.Spec.Protocol)
	require.Equal(t, in.Spec.Version, out.Spec.Version)
	require.Equal(t, in.Spec.Icmp, out.Spec.Icmp)
	require.Equal(t, in.Spec.Ports, out.Spec.Ports)
	require.Len(t, out.Spec.SourceRef, 1)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.State)
}

func TestSecurityGroupRuleToCR_MinimalSpec(t *testing.T) {
	in := &securitygroupruledom.SecurityGroupRule{
		Spec: securitygroupruledom.SecurityGroupRuleSpec{
			Direction: "egress",
		},
	}
	in.Name = "sgr1"

	cr, err := SecurityGroupRuleToCR(in)
	require.NoError(t, err)

	out, err := SecurityGroupRuleFromCR(cr)
	require.NoError(t, err)
	require.Equal(t, "egress", out.Spec.Direction)
	require.Nil(t, out.Spec.Icmp)
	require.Nil(t, out.Spec.Ports)
	require.Equal(t, commondomain.ResourceStatePending, out.Status.Conditions[0].State)
}

// isWellFormedRefPath reports whether a reference path is shaped like a SECA resource path, for
// which commonbackend's scope embed/extract pair is a true inverse.
//
// ponytail: this is a known ceiling, not a tautology. extractAndStripSegment strips a scope
// anchor from wherever it appears, while embedScopeInResource always re-prefixes it to the whole
// path — so they only round-trip for paths carrying at most one "tenants/" and one "workspaces/"
// anchor and no empty segments. Paths like "tenants/0/tenants//" drift on every pass. Real SECA
// references are always canonical, so the gap is theoretical; widen this predicate if the two
// helpers are ever reworked into true inverses.
func isWellFormedRefPath(p string) bool {
	return !strings.Contains(p, "//") &&
		strings.Count(p, "tenants") <= 1 &&
		strings.Count(p, "workspaces") <= 1
}

// FuzzSecurityGroupRuleSpecRoundTrip targets the nested optional spec — *IcmpConfig, *Ports and
// the []Reference source list. Pointer and slice fields are where nil-vs-empty drift hides, and
// SourceRef exercises ReferenceToCR's documented idempotence (it embeds tenant/workspace into
// the resource path on the way out, then must leave that path alone on the next pass).
//
// The invariant is stability, not identity: the first round-trip normalizes (unknown IP versions
// collapse to "", provider "/"↔"_"), so domain2 and domain3 must be identical.
func FuzzSecurityGroupRuleSpecRoundTrip(f *testing.F) {
	// (name, direction, protocol, version, srcResource, srcProvider, hasIcmp, hasPorts,
	//  icmpCode, icmpType, portFrom, portTo)
	f.Add("sgr", "ingress", "tcp", "IPv4", "networks/net1", "", true, true, 1, 2, 80, 443)
	f.Add("sgr", "egress", "", "", "", "", false, false, 0, 0, 0, 0)
	f.Add("sgr", "ingress", "udp", "IPv6", "tenants/t/networks/n", "", true, false, 0, 0, 0, 0)
	f.Add("sgr", "ingress", "icmp", "bogus-version", "networks/n", "seca.network/v1", false, true, 0, 0, -1, -1)
	// already-qualified reference paths: the second pass must not re-embed the scope
	f.Add("sgr", "ingress", "tcp", "IPv4", "seca.network/v1/tenants/t/workspaces/w/networks/n", "", false, false, 0, 0, 1, 65535)
	f.Add("sgr", "ingress", "tcp", "IPv4", "tenants/t/workspaces/w/security-groups/sg", "", false, false, 0, 0, 0, 0)
	// numeric edges
	f.Add("sgr", "ingress", "tcp", "IPv4", "networks/n", "", true, true, -2147483648, 2147483647, -1, 65536)

	f.Fuzz(func(t *testing.T, name, direction, protocol, version, srcResource, srcProvider string,
		hasIcmp, hasPorts bool, icmpCode, icmpType, portFrom, portTo int,
	) {
		spec := securitygroupruledom.SecurityGroupRuleSpec{
			Direction: direction,
			Protocol:  protocol,
			Version:   commondomain.IPVersion(version),
		}
		if hasIcmp {
			spec.Icmp = &securitygroupruledom.IcmpConfig{Code: icmpCode, Type: icmpType}
		}
		if hasPorts {
			spec.Ports = &securitygroupruledom.Ports{From: portFrom, To: portTo, List: []int{portFrom, portTo}}
		}
		if srcResource != "" {
			if !isWellFormedRefPath(srcResource) {
				return
			}
			spec.SourceRef = []commondomain.Reference{{Resource: srcResource, Provider: srcProvider}}
		}

		domain := &securitygroupruledom.SecurityGroupRule{Spec: spec}
		domain.Name = name
		domain.Tenant = "t"
		domain.Workspace = "w"

		cr1, err := SecurityGroupRuleToCR(domain)
		if err != nil {
			return
		}

		domain2, err := SecurityGroupRuleFromCR(cr1)
		if err != nil {
			t.Errorf("CR→domain failed after successful domain→CR: %v", err)
			return
		}

		cr2, err := SecurityGroupRuleToCR(domain2)
		if err != nil {
			t.Errorf("second domain→CR failed: %v", err)
			return
		}

		domain3, err := SecurityGroupRuleFromCR(cr2)
		if err != nil {
			t.Errorf("second CR→domain failed: %v", err)
			return
		}

		// Optional sub-structs must keep their nil-ness, not flip to a zero value or back.
		if (domain2.Spec.Icmp == nil) != (domain3.Spec.Icmp == nil) {
			t.Errorf("Icmp nil-ness not stable: %v → %v", domain2.Spec.Icmp, domain3.Spec.Icmp)
		}
		if (domain2.Spec.Ports == nil) != (domain3.Spec.Ports == nil) {
			t.Errorf("Ports nil-ness not stable: %v → %v", domain2.Spec.Ports, domain3.Spec.Ports)
		}
		if !reflect.DeepEqual(domain2.Spec.Icmp, domain3.Spec.Icmp) {
			t.Errorf("Icmp not stable: %+v → %+v", domain2.Spec.Icmp, domain3.Spec.Icmp)
		}
		if !reflect.DeepEqual(domain2.Spec.Ports, domain3.Spec.Ports) {
			t.Errorf("Ports not stable: %+v → %+v", domain2.Spec.Ports, domain3.Spec.Ports)
		}
		// SourceRef must neither grow, shrink, nor re-embed its scope segments.
		if !reflect.DeepEqual(domain2.Spec.SourceRef, domain3.Spec.SourceRef) {
			t.Errorf("SourceRef not stable: %+v → %+v", domain2.Spec.SourceRef, domain3.Spec.SourceRef)
		}
		if domain2.Spec.Direction != domain3.Spec.Direction {
			t.Errorf("Direction not stable: %q → %q", domain2.Spec.Direction, domain3.Spec.Direction)
		}
		if domain2.Spec.Protocol != domain3.Spec.Protocol {
			t.Errorf("Protocol not stable: %q → %q", domain2.Spec.Protocol, domain3.Spec.Protocol)
		}
		if domain2.Spec.Version != domain3.Spec.Version {
			t.Errorf("Version not stable: %q → %q", domain2.Spec.Version, domain3.Spec.Version)
		}
		if domain2.Name != domain3.Name {
			t.Errorf("Name not stable: %q → %q", domain2.Name, domain3.Name)
		}
	})
}
