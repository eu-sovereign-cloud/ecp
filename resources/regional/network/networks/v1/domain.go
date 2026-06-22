// Package domain defines the network resource domain model and identity constants.
package v1

import "github.com/eu-sovereign-cloud/ecp/resources/common/domain"

// Identity constants for the network resource.
const (
	Kind       = "Network"
	Resource   = "networks"
	Group      = "network.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.network/v1"
)

// NetworkDomain represents the domain model for a network instance.
type NetworkDomain struct {
	domain.RegionalMetadata
	Spec   NetworkSpecDomain
	Status *NetworkStatusDomain
}

// NetworkSpecDomain defines the specification for a network instance.
type NetworkSpecDomain struct {
	AdditionalCidrs []CidrDomain
	Cidr            CidrDomain
	SkuRef          domain.ReferenceDomain
	RouteTableRef   domain.ReferenceDomain
}

// CidrDomain holds IPv4 and IPv6 CIDR strings for a network address range.
// Either field may be empty: IPv4-only, IPv6-only, or dual-stack.
type CidrDomain struct {
	IPv4 string
	IPv6 string
}

// NetworkStatusDomain defines the status for a network instance.
type NetworkStatusDomain struct {
	domain.StatusDomain
	// TODO: add Cidr/AdditionalCidrs/RouteTableRef from SECA NetworkStatus when the reconciler surfaces assigned ranges
}
