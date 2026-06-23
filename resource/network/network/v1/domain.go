// Package v1 defines the network resource domain model and identity constants.
package v1

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the network resource.
const (
	Kind       = "Network"
	Resource   = "networks"
	Group      = "network.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.network/v1"
)

// Network represents the domain model for a network instance.
type Network struct {
	domain.RegionalMetadata
	Spec   NetworkSpec
	Status *NetworkStatus
}

// NetworkSpec defines the specification for a network instance.
type NetworkSpec struct {
	AdditionalCidrs []Cidr
	Cidr            Cidr
	SkuRef          domain.ReferenceDomain
	RouteTableRef   domain.ReferenceDomain
}

// Cidr holds IPv4 and IPv6 CIDR strings for a network address range.
// Either field may be empty: IPv4-only, IPv6-only, or dual-stack.
type Cidr struct {
	IPv4 string
	IPv6 string
}

// NetworkStatus defines the status for a network instance.
type NetworkStatus struct {
	domain.StatusDomain
	// TODO: add Cidr/AdditionalCidrs/RouteTableRef from SECA NetworkStatus when the reconciler surfaces assigned ranges
}
