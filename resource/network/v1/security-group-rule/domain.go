// Package securitygrouprule defines the security group rule resource domain model and identity constants.
package securitygrouprule

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the SecurityGroupRule resource.
const (
	Kind       = "SecurityGroupRule"
	Resource   = "security-group-rules"
	Group      = "network.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.network/v1"
)

// SecurityGroupRule represents the domain model for a security group rule.
type SecurityGroupRule struct {
	domain.RegionalMetadata
	Spec   SecurityGroupRuleSpec
	Status *SecurityGroupRuleStatus
}

// SecurityGroupRuleSpec defines the specification for a SecurityGroupRule.
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

// SecurityGroupRuleStatus defines the status for a SecurityGroupRule. It carries no
// resource-specific fields — mirrors network.NetworkStatus.
type SecurityGroupRuleStatus struct {
	domain.Status
}
