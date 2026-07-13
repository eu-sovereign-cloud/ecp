// Package subnet defines the subnet resource domain model and identity constants.
package subnet

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the Subnet resource.
const (
	Kind       = "Subnet"
	Resource   = "subnets"
	Group      = "network.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.network/v1"
)

// Subnet represents the domain model for a subnet.
type Subnet struct {
	domain.RegionalNetworkMetadata
	Spec   SubnetSpec
	Status *SubnetStatus
}

// SubnetSpec defines the specification for a Subnet.
type SubnetSpec struct {
	Cidr          CIDR
	RouteTableRef domain.Reference
	// SkuRef is optional and immutable. The zero value means unset.
	SkuRef domain.Reference
	Zone   string
}

// CIDR holds IPv4 and IPv6 CIDR strings for a subnet address range.
// Either field may be empty: IPv4-only, IPv6-only, or dual-stack.
type CIDR struct {
	IPv4 string
	IPv6 string
}

// SubnetStatus defines the status for a Subnet.
type SubnetStatus struct {
	domain.Status
	Cidr *CIDR
	// RouteTableRef is the route table actually in effect for this subnet (its own, or the
	// network's, when unset on the spec). Nil until assigned.
	RouteTableRef *domain.Reference
}
