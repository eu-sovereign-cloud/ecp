// Package securitygroup defines the security group resource domain model and identity constants.
package securitygroup

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the SecurityGroup resource.
const (
	Kind       = "SecurityGroup"
	Resource   = "security-groups"
	Group      = "network.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.network/v1"
)

// SecurityGroup represents the domain model for a security group.
type SecurityGroup struct {
	domain.RegionalMetadata
	Spec   SecurityGroupSpec
	Status *SecurityGroupStatus
}

// SecurityGroupSpec defines the specification for a SecurityGroup.
type SecurityGroupSpec struct {
	// RuleRefs references shared SecurityGroupRule resources, applied in addition to Rules.
	RuleRefs []domain.Reference
	// Rules are inline network access rules. Default behavior is to deny all traffic not
	// explicitly allowed.
	Rules []SecurityGroupRuleSpec
}

// SecurityGroupRuleSpec defines a single inline rule of a SecurityGroup.
type SecurityGroupRuleSpec struct {
	// Direction of the traffic flow: "ingress" or "egress".
	Direction string
	// Icmp carries ICMP-specific rule configuration. Nil unless Protocol is icmp.
	Icmp *IcmpConfig
	// Ports defines a specific port list or range for the rule.
	Ports *Ports
	// Protocol for the rule. Empty means any protocol is allowed.
	Protocol string
	// SourceRef references the CIDR block, IP address, gateway, instance or security group
	// allowed to communicate with the security group. Empty means all traffic is allowed.
	SourceRef []domain.Reference
	// Version restricts the rule to an IP version. Empty means any version is allowed.
	Version domain.IPVersion
}

// IcmpConfig defines ICMP-specific rule configuration.
type IcmpConfig struct {
	Code int
	Type int
}

// Ports defines a specific port list or range for a rule.
type Ports struct {
	From int
	List []int
	To   int
}

// SecurityGroupStatus defines the status for a SecurityGroup.
type SecurityGroupStatus struct {
	domain.Status
	// Rules carries per-rule status; left empty by the dummy plugin, which only manages
	// whole-resource state.
	Rules []SecurityGroupRuleStatus
}

// SecurityGroupRuleStatus defines the status of a single inline rule within a SecurityGroup.
type SecurityGroupRuleStatus struct {
	State      domain.ResourceState
	Conditions []domain.StatusCondition
}
