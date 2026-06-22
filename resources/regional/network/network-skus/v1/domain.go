// Package domain defines the network SKU resource domain model and identity constants.
package v1

import "github.com/eu-sovereign-cloud/ecp/resources/common/domain"

// Identity constants for the network SKU resource.
const (
	Kind       = "NetworkSKU"
	Resource   = "network-skus"
	Group      = "network.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.network/v1"
)

// NetworkSKU represents the domain model for a network SKU.
type NetworkSKU struct {
	domain.RegionalMetadata
	Spec NetworkSKUSpec
}

// NetworkSKUSpec defines the specification for a network SKU.
type NetworkSKUSpec struct {
	Bandwidth int
	Packets   int
}
