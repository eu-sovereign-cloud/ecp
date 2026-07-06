// Package nic defines the NIC resource domain model and identity constants.
package nic

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the NIC resource.
const (
	Kind       = "NIC"
	Resource   = "nics"
	Group      = "network.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.network/v1"
)

// Nic represents the domain model for a network interface card.
type Nic struct {
	domain.RegionalMetadata
	Spec   NicSpec
	Status *NicStatus
}

// NicSpec defines the specification for a NIC.
type NicSpec struct {
	Addresses         []string
	PublicIpRefs      []domain.Reference
	SecurityGroupRefs []domain.Reference
	// SkuRef is optional and immutable. The zero value means unset.
	SkuRef    domain.Reference
	SubnetRef domain.Reference
}

// NicStatus defines the status for a NIC.
type NicStatus struct {
	domain.Status
	Addresses    []string
	MacAddress   string
	PublicIpRefs []domain.Reference
}
