package regional

// NetworkDomain represents the domain model for a network instance.
type NetworkDomain struct {
	Metadata
	Spec   NetworkSpecDomain
	Status *NetworkStatusDomain
}

// NetworkSpecDomain defines the specification for a network instance.
type NetworkSpecDomain struct {
	AdditionalCidrs []CidrDomain
	Cidr            CidrDomain
	SkuRef          ReferenceDomain
	RouteTableRef   ReferenceDomain
}

// CidrDomain holds IPv4 and IPv6 CIDR strings for a network address range.
// Either field may be empty: IPv4-only, IPv6-only, or dual-stack.
type CidrDomain struct {
	IPv4 string
	IPv6 string
}

// NetworkStatusDomain defines the status for a network instance.
type NetworkStatusDomain struct {
	StatusDomain
	// TODO: add Cidr/AdditionalCidrs/RouteTableRef from SECA NetworkStatus when the reconciler surfaces assigned ranges
}
