package converter

import (
	"fmt"
	"strings"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroup "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	securitygrouprule "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

// A SECA SecurityGroup carries no VPC and takes effect only once an instance/NIC attaches it, so
// the Aruba SecurityGroup (which requires a VPCReference) cannot be created at SECA-SG creation
// time. The compute-instance handler materialises it here, once per (network, SECA security
// group), in the VPC the attaching instance lives in. See csp/aruba/README.md.

// Labels stamped on every materialised Aruba SecurityGroup and SecurityRule. They are the only
// link back to the SECA security group, so the SecurityGroupHandler reaps by these keys when the
// SECA group is deleted - keep the two writers below and that reader in step through these consts.
// Note LabelSecurityGroup holds the SECA group name on a SecurityGroup but the materialised group
// name on a SecurityRule (a rule belongs to one materialised group, not directly to the SECA one).
const (
	LabelTenant        = "seca.securitygroup/tenant"
	LabelNetwork       = "seca.securitygroup/network"
	LabelSecurityGroup = "seca.securitygroup/securitygroup"
)

// MaterializedSecurityGroupName is the name of the Aruba SecurityGroup that backs a SECA security
// group inside one network's VPC. The network is encoded because the same SECA security group may
// be attached in several networks, each needing its own Aruba SecurityGroup in that VPC.
func MaterializedSecurityGroupName(secaName, network string) string {
	return fmt.Sprintf("%s-%s", secaName, network)
}

// BuildSecurityGroup maps a SECA security group to an Aruba SecurityGroup in a given VPC.
func BuildSecurityGroup(secaName, network, region, tenant, namespace string, vpcRef, projectRef v1alpha1.ResourceReference) *v1alpha1.SecurityGroup {
	if region == "" {
		region = defaultRegion
	}

	return &v1alpha1.SecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MaterializedSecurityGroupName(secaName, network),
			Namespace: namespace,
			Labels: map[string]string{
				LabelTenant:        tenant,
				LabelNetwork:       network,
				LabelSecurityGroup: secaName,
			},
		},
		Spec: v1alpha1.SecurityGroupSpec{
			Tenant:           tenant,
			Region:           region,
			VPCReference:     vpcRef,
			ProjectReference: projectRef,
		},
	}
}

// RuleSpec is the provider-neutral shape of a SECA rule, normalised from either an inline
// SecurityGroup rule or a standalone SecurityGroupRule so BuildSecurityRules can expand both.
type RuleSpec struct {
	Direction string
	Protocol  string
	PortFrom  int
	PortTo    int
	PortList  []int
	HasIcmp   bool
	SourceRef []commondomain.Reference
}

// NormalizeInlineRules converts a SECA SecurityGroup's inline rules to RuleSpecs.
func NormalizeInlineRules(rules []securitygroup.SecurityGroupRuleSpec) []RuleSpec {
	out := make([]RuleSpec, 0, len(rules))
	for _, r := range rules {
		spec := RuleSpec{Direction: r.Direction, Protocol: r.Protocol, SourceRef: r.SourceRef}
		if r.Ports != nil {
			spec.PortFrom, spec.PortTo, spec.PortList = r.Ports.From, r.Ports.To, r.Ports.List
		}
		spec.HasIcmp = r.Icmp != nil
		out = append(out, spec)
	}
	return out
}

// NormalizeStandaloneRule converts a standalone SECA SecurityGroupRule to a RuleSpec.
func NormalizeStandaloneRule(r securitygrouprule.SecurityGroupRuleSpec) RuleSpec {
	spec := RuleSpec{Direction: r.Direction, Protocol: r.Protocol, SourceRef: r.SourceRef}
	if r.Ports != nil {
		spec.PortFrom, spec.PortTo, spec.PortList = r.Ports.From, r.Ports.To, r.Ports.List
	}
	spec.HasIcmp = r.Icmp != nil
	return spec
}

// BuildSecurityRules expands normalised SECA rules into Aruba SecurityRules. An Aruba SecurityRule
// carries a single protocol, a single port/range and a single target, so one SECA rule fans out
// across (protocol x port x source): a "tcp+udp" rule yields TCP and UDP rules, a port list yields
// one rule per port, and each sourceRef yields its own rule. Rules are named deterministically so
// re-issuing the create is idempotent.
func BuildSecurityRules(rules []RuleSpec, sgName, region, tenant, namespace string, vpcRef, projectRef v1alpha1.ResourceReference) []*v1alpha1.SecurityRule {
	if region == "" {
		region = defaultRegion
	}

	sgRef := v1alpha1.ResourceReference{Name: sgName, Namespace: namespace}

	var out []*v1alpha1.SecurityRule
	idx := 0
	for _, r := range rules {
		direction := arubaDirection(r.Direction)
		for _, protocol := range arubaProtocols(r.Protocol, r.HasIcmp) {
			for _, port := range arubaPorts(r, protocol) {
				for _, target := range arubaTargets(r.SourceRef) {
					out = append(out, &v1alpha1.SecurityRule{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-r%d", sgName, idx),
							Namespace: namespace,
							Labels: map[string]string{
								LabelTenant:        tenant,
								LabelSecurityGroup: sgName,
							},
						},
						Spec: v1alpha1.SecurityRuleSpec{
							Tenant:                 tenant,
							Region:                 region,
							Protocol:               protocol,
							Port:                   port,
							Direction:              direction,
							Target:                 target,
							SecurityGroupReference: sgRef,
							VPCReference:           vpcRef,
							ProjectReference:       projectRef,
						},
					})
					idx++
				}
			}
		}
	}
	return out
}

func arubaDirection(direction string) string {
	if strings.EqualFold(direction, "egress") {
		return "Egress"
	}
	return "Ingress"
}

// arubaProtocols maps a SECA protocol to Aruba's enum (TCP;UDP;ICMP;ALL). "tcp+udp" has no single
// Aruba equivalent and expands to two protocols; an empty protocol means "any" -> ALL.
func arubaProtocols(protocol string, hasIcmp bool) []string {
	switch strings.ToLower(protocol) {
	case "tcp":
		return []string{"TCP"}
	case "udp":
		return []string{"UDP"}
	case "tcp+udp":
		return []string{"TCP", "UDP"}
	case "icmp":
		return []string{"ICMP"}
	case "":
		if hasIcmp {
			return []string{"ICMP"}
		}
		return []string{"ALL"}
	default:
		return []string{strings.ToUpper(protocol)}
	}
}

// arubaPorts maps a SECA rule's ports to Aruba port strings. ICMP carries no ports; a port list
// yields one entry per port; a from/to pair yields a range (or a single port when they coincide);
// otherwise the rule matches every port -> ALL.
func arubaPorts(r RuleSpec, protocol string) []string {
	if protocol == "ICMP" {
		return []string{"ALL"}
	}
	if len(r.PortList) > 0 {
		ports := make([]string, 0, len(r.PortList))
		for _, p := range r.PortList {
			ports = append(ports, fmt.Sprintf("%d", p))
		}
		return ports
	}
	if r.PortFrom > 0 || r.PortTo > 0 {
		if r.PortTo == 0 || r.PortFrom == r.PortTo {
			return []string{fmt.Sprintf("%d", r.PortFrom)}
		}
		return []string{fmt.Sprintf("%d-%d", r.PortFrom, r.PortTo)}
	}
	return []string{"ALL"}
}

// arubaTargets maps a SECA rule's sourceRefs to Aruba rule targets. A security-group reference
// becomes a SecurityGroup target; anything else is treated as an IP/CIDR literal. An empty
// sourceRef means "all traffic" -> 0.0.0.0/0. Instance/gateway sources have no Aruba target type
// and are mapped best-effort to Ip (see csp/aruba/README.md).
func arubaTargets(sources []commondomain.Reference) []v1alpha1.SecurityRuleTarget {
	if len(sources) == 0 {
		return []v1alpha1.SecurityRuleTarget{{Type: "Ip", Value: "0.0.0.0/0"}}
	}

	targets := make([]v1alpha1.SecurityRuleTarget, 0, len(sources))
	for _, s := range sources {
		if name, ok := strings.CutPrefix(s.Resource, "security-groups/"); ok {
			targets = append(targets, v1alpha1.SecurityRuleTarget{Type: "SecurityGroup", Value: name})
			continue
		}
		targets = append(targets, v1alpha1.SecurityRuleTarget{Type: "Ip", Value: s.Resource})
	}
	return targets
}
