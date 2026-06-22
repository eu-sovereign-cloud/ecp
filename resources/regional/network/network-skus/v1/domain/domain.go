// Package domain defines the network SKU resource domain model and identity constants.
package domain

import "github.com/eu-sovereign-cloud/ecp/resources/common/domain"

// Identity constants for the network SKU resource.
const (
	Kind       = "NetworkSKU"
	Resource   = "network-skus"
	Group      = "network.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.network/v1"
)

// NetworkSKUDomain represents the domain model for a network SKU.
type NetworkSKUDomain struct {
	domain.RegionalMetadata
	Spec NetworkSKUSpecDomain
}

// NetworkSKUSpecDomain defines the specification for a network SKU.
type NetworkSKUSpecDomain struct {
	Bandwidth int
	Packets   int
}
