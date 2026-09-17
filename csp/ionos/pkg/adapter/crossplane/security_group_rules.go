package crossplane

import (
	"crypto/sha256"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

// IONOS firewall rule directions.
const (
	firewallTypeIngress = "INGRESS"
	firewallTypeEgress  = "EGRESS"
)

// IONOS NSG firewall rule protocols. The NSG rule enum is narrower than the classic NIC
// firewall's (no ICMPv6/GRE/VRRP/ESP/AH), but it covers everything SECA's own closed enum
// (tcp, udp, tcp+udp, icmp) can ask for.
const (
	protocolTCP  = "TCP"
	protocolUDP  = "UDP"
	protocolICMP = "ICMP"
	protocolAny  = "ANY"
)

// LabelSecurityGroup ties every NSGFirewallRule CR back to the SECA security group that
// produced it. It is how the store finds a group's rules without rebuilding the names the
// current spec would produce — which is what lets a reconcile reap a rule the spec has dropped,
// and a delete reap rules left over from an earlier spec.
const LabelSecurityGroup = "seca.ionos/security-group"

// normalizedRule is the provider-neutral shape of a SECA rule. A security group's inline rules
// and a standalone SecurityGroupRule are structurally identical but are distinct Go types, so
// both are normalised to this one shape and a single expansion covers them.
type normalizedRule struct {
	Direction string
	Protocol  string
	PortFrom  int
	PortTo    int
	PortList  []int
	Icmp      *icmpConfig
	SourceRef []commondomain.Reference
	Version   commondomain.IPVersion
	// Origin names the rule in error messages: the owning group for an inline rule, the
	// standalone rule's own name otherwise.
	Origin string
}

type icmpConfig struct {
	Code int
	Type int
}

// normalizeInlineRules converts a security group's inline rules to normalizedRules.
func normalizeInlineRules(rules []securitygroupdom.SecurityGroupRuleSpec, origin string) []normalizedRule {
	out := make([]normalizedRule, 0, len(rules))
	for _, r := range rules {
		n := normalizedRule{
			Direction: r.Direction,
			Protocol:  r.Protocol,
			SourceRef: r.SourceRef,
			Version:   r.Version,
			Origin:    origin,
		}
		if r.Ports != nil {
			n.PortFrom, n.PortTo, n.PortList = r.Ports.From, r.Ports.To, r.Ports.List
		}
		if r.Icmp != nil {
			n.Icmp = &icmpConfig{Code: r.Icmp.Code, Type: r.Icmp.Type}
		}
		out = append(out, n)
	}
	return out
}

// normalizeStandaloneRule converts a standalone SECA SecurityGroupRule to a normalizedRule.
func normalizeStandaloneRule(r securitygroupruledom.SecurityGroupRuleSpec, origin string) normalizedRule {
	n := normalizedRule{
		Direction: r.Direction,
		Protocol:  r.Protocol,
		SourceRef: r.SourceRef,
		Version:   r.Version,
		Origin:    origin,
	}
	if r.Ports != nil {
		n.PortFrom, n.PortTo, n.PortList = r.Ports.From, r.Ports.To, r.Ports.List
	}
	if r.Icmp != nil {
		n.Icmp = &icmpConfig{Code: r.Icmp.Code, Type: r.Icmp.Type}
	}
	return n
}

// portRange is one IONOS port window. A nil slice from expandPorts means "every port".
type portRange struct {
	From int
	To   int
}

// expandPorts turns a SECA rule's ports into IONOS port windows.
//
// Per the SECA schema, `from`/`to` describe one range (either bound alone meaning that single
// port) and `list` adds *further individual ports* on top of it — the two are additive, not
// alternatives. An IONOS rule carries a single window, so each one becomes its own rule.
// No ports at all means the rule matches every port, which IONOS expresses as an unset range.
func expandPorts(from, to int, list []int) []portRange {
	var out []portRange
	switch {
	case from > 0 && to > 0:
		out = append(out, portRange{From: from, To: to})
	case from > 0:
		out = append(out, portRange{From: from, To: from})
	case to > 0:
		out = append(out, portRange{From: to, To: to})
	}
	for _, p := range list {
		out = append(out, portRange{From: p, To: p})
	}
	return out
}

// expandProtocols maps a SECA protocol to the IONOS enum. "tcp+udp" has no single IONOS
// equivalent and expands to two rules; an empty protocol means "any", except on a rule carrying
// an icmp block, where it means ICMP.
func expandProtocols(protocol string, hasIcmp bool) ([]string, error) {
	switch strings.ToLower(protocol) {
	case "tcp":
		return []string{protocolTCP}, nil
	case "udp":
		return []string{protocolUDP}, nil
	case "tcp+udp":
		return []string{protocolTCP, protocolUDP}, nil
	case "icmp":
		return []string{protocolICMP}, nil
	case "":
		if hasIcmp {
			return []string{protocolICMP}, nil
		}
		return []string{protocolAny}, nil
	default:
		return nil, fmt.Errorf("%w: protocol %q has no IONOS NSG equivalent (IONOS accepts TCP, UDP, ICMP or ANY)",
			backend.ErrNotSupported, protocol)
	}
}

// firewallType maps a SECA direction to the IONOS rule type. IONOS defaults an unset type to
// INGRESS, and so does SECA, so anything that is not egress is ingress.
func firewallType(direction string) string {
	if strings.EqualFold(direction, "egress") {
		return firewallTypeEgress
	}
	return firewallTypeIngress
}

// expandSources turns a rule's sourceRefs into IONOS sourceIp values. An empty sourceRef means
// "all traffic", which IONOS expresses as an unset sourceIp — represented here as a single nil
// entry so the caller still emits exactly one rule.
//
// An IONOS NSG rule matches only on a source address (or CIDR), so a SECA sourceRef naming a
// security group, instance or gateway cannot be expressed and is refused rather than quietly
// dropped or mapped onto something narrower.
func expandSources(rule normalizedRule) ([]*string, error) {
	if len(rule.SourceRef) == 0 {
		if rule.Version != "" {
			// IONOS carries an ipVersion on a firewall rule, but the Crossplane
			// NSGFirewallRule does not expose it, so the family can only be implied by the
			// source address. With no source, IONOS defaults the rule to IPv4 only —
			// silently ignoring an explicit "IPv6" would hand back a rule that does not do
			// what was asked.
			return nil, fmt.Errorf("%w: rule %q sets version %q with no sourceRef; the Crossplane NSGFirewallRule exposes no ipVersion field, so the family can only be implied by a source address",
				backend.ErrNotSupported, rule.Origin, rule.Version)
		}
		return []*string{nil}, nil
	}

	out := make([]*string, 0, len(rule.SourceRef))
	for _, ref := range rule.SourceRef {
		literal, err := sourceLiteral(ref, rule.Origin)
		if err != nil {
			return nil, err
		}
		if err := checkIPVersion(literal, rule); err != nil {
			return nil, err
		}
		out = append(out, new(literal))
	}
	return out, nil
}

// sourceLiteral reads a sourceRef as an IP address or CIDR block. Anything that does not parse
// as one is a reference to another SECA resource, which an IONOS NSG rule cannot target.
func sourceLiteral(ref commondomain.Reference, origin string) (string, error) {
	res := ref.Resource
	if _, err := netip.ParsePrefix(res); err == nil {
		return res, nil
	}
	if _, err := netip.ParseAddr(res); err == nil {
		return res, nil
	}
	return "", fmt.Errorf("%w: rule %q has sourceRef %q, which is not an IP address or CIDR block; an IONOS NSG rule matches only on source address, so a security group, instance or gateway source cannot be expressed",
		backend.ErrNotSupported, origin, res)
}

// checkIPVersion rejects a rule whose declared version contradicts its source address. The
// version cannot be sent to IONOS (no ipVersion field on the CR) so it is only ever implied by
// the source — a contradiction would therefore apply a rule for the wrong family in silence.
func checkIPVersion(literal string, rule normalizedRule) error {
	if rule.Version == "" {
		return nil
	}
	is4, err := literalIsIPv4(literal)
	if err != nil {
		return err
	}
	want4 := rule.Version == commondomain.IPVersionIPv4
	if is4 != want4 {
		return fmt.Errorf("%w: rule %q declares version %q but sourceRef %q is of the other family; IONOS derives a rule's IP version from its source address, so the two cannot disagree",
			backend.ErrNotSupported, rule.Origin, rule.Version, literal)
	}
	return nil
}

func literalIsIPv4(literal string) (bool, error) {
	if prefix, err := netip.ParsePrefix(literal); err == nil {
		return prefix.Addr().Is4(), nil
	}
	addr, err := netip.ParseAddr(literal)
	if err != nil {
		return false, fmt.Errorf("parse source %q: %w", literal, err)
	}
	return addr.Is4(), nil
}

// buildRuleCRs expands normalised SECA rules into IONOS NSGFirewallRule CRs.
//
// An IONOS rule carries a single protocol, a single port window and a single source, so one
// SECA rule fans out across (protocol x port x source): "tcp+udp" yields TCP and UDP rules, each
// port window yields its own rule, and each sourceRef yields its own rule.
func buildRuleCRs(sgName, workspace, namespace, datacenterNamespace string, rules []normalizedRule) ([]*ionosv1alpha1.NSGFirewallRule, error) {
	var out []*ionosv1alpha1.NSGFirewallRule

	for _, rule := range rules {
		protocols, err := expandProtocols(rule.Protocol, rule.Icmp != nil)
		if err != nil {
			return nil, err
		}
		sources, err := expandSources(rule)
		if err != nil {
			return nil, err
		}
		ruleType := firewallType(rule.Direction)

		for _, protocol := range protocols {
			// IONOS accepts a port range only on TCP and UDP; ICMP and ANY carry none.
			windows := []portRange(nil)
			if protocol == protocolTCP || protocol == protocolUDP {
				windows = expandPorts(rule.PortFrom, rule.PortTo, rule.PortList)
			}
			// A nil window list still emits one rule — one that matches every port.
			if len(windows) == 0 {
				windows = []portRange{{}}
			}

			for _, window := range windows {
				for _, source := range sources {
					params := nsgRuleParams{
						SecurityGroup:       sgName,
						Workspace:           workspace,
						Namespace:           namespace,
						DatacenterNamespace: datacenterNamespace,
						Protocol:            protocol,
						Type:                ruleType,
						Window:              window,
						SourceIP:            source,
						Icmp:                rule.Icmp,
					}
					params.Name = ruleCRName(sgName, params)
					out = append(out, newNSGFirewallRule(params))
				}
			}
		}
	}
	return out, nil
}

// ruleCRName derives a rule CR's name from what the rule *says*, not from its position in the
// spec.
//
// Content addressing is what lets a group's rules be reconciled without ever rewriting a rule in
// place. Editing a rule changes its name, so the reconcile reaps the CR carrying the old content
// and creates one for the new — which is also what IONOS needs, since a rule's protocol is
// immutable and an in-place change would be rejected outright. Rewriting instead would mean
// diffing against fields the provider late-initialises (sourceIp, targetIp and type are all
// computed), and a mismatch there would write on every pass and never converge.
//
// Position-derived names cannot do this: an edited rule would keep its name, the create would
// come back AlreadyExists, and the edit would be dropped in silence. Two further properties fall
// out — reordering a group's rules churns nothing, and two rules that say the same thing collapse
// onto one CR, which is the same policy either way.
func ruleCRName(sgName string, p nsgRuleParams) string {
	var icmpType, icmpCode string
	if p.Icmp != nil {
		icmpType, icmpCode = strconv.Itoa(p.Icmp.Type), strconv.Itoa(p.Icmp.Code)
	}
	source := ""
	if p.SourceIP != nil {
		source = *p.SourceIP
	}

	// NUL-separated so no field's content can spill into the next one's.
	sum := sha256.Sum256([]byte(strings.Join([]string{
		p.Type, p.Protocol,
		strconv.Itoa(p.Window.From), strconv.Itoa(p.Window.To),
		source, icmpType, icmpCode,
	}, "\x00")))

	return fmt.Sprintf("%s-%x", sgName, sum[:6])
}

type nsgRuleParams struct {
	Name                string
	SecurityGroup       string
	Workspace           string
	Namespace           string
	DatacenterNamespace string
	Protocol            string
	Type                string
	Window              portRange
	SourceIP            *string
	Icmp                *icmpConfig
}

func newNSGFirewallRule(p nsgRuleParams) *ionosv1alpha1.NSGFirewallRule {
	rule := &ionosv1alpha1.NSGFirewallRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ionosv1alpha1.CRDGroupVersion.String(),
			Kind:       ionosv1alpha1.NSGFirewallRule_Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
			Labels:    map[string]string{LabelSecurityGroup: p.SecurityGroup},
		},
		Spec: ionosv1alpha1.NSGFirewallRuleSpec{
			ForProvider: ionosv1alpha1.NSGFirewallRuleParameters{
				Name:     new(p.Name),
				Protocol: new(p.Protocol),
				Type:     new(p.Type),
				NsgIDRef: &v1.NamespacedReference{Name: p.SecurityGroup, Namespace: p.Namespace},
				DatacenterIDRef: &v1.NamespacedReference{
					Name:      p.Workspace,
					Namespace: p.DatacenterNamespace,
				},
			},
			ManagedResourceSpec: providerConfig(),
		},
	}

	if p.Window.From > 0 {
		rule.Spec.ForProvider.PortRangeStart = new(float64(p.Window.From))
		rule.Spec.ForProvider.PortRangeEnd = new(float64(p.Window.To))
	}
	if p.SourceIP != nil {
		rule.Spec.ForProvider.SourceIP = new(*p.SourceIP)
	}
	// ICMP type/code are only meaningful on an ICMP rule. Left unset they mean "all types"
	// and "all codes"; SECA's IcmpConfig has no unset marker (0 is a valid type and code),
	// so an absent icmp block is what selects the permissive default.
	if p.Protocol == protocolICMP && p.Icmp != nil {
		rule.Spec.ForProvider.IcmpType = new(strconv.Itoa(p.Icmp.Type))
		rule.Spec.ForProvider.IcmpCode = new(strconv.Itoa(p.Icmp.Code))
	}

	return rule
}
