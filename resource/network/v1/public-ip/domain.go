// Package publicip defines the public IP resource domain model and identity constants.
package publicip

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the PublicIp resource.
const (
	Kind       = "PublicIP"
	Resource   = "public-ips"
	Group      = "network.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.network/v1"
)

// PublicIp represents the domain model for a public IP address.
type PublicIp struct {
	domain.RegionalMetadata
	Spec   PublicIpSpec
	Status *PublicIpStatus
}

// PublicIpSpec defines the specification for a PublicIp.
type PublicIpSpec struct {
	// Address is optional (BYOIP). Empty means unset.
	Address string
	Version domain.IPVersion
}

// PublicIpStatus defines the status for a PublicIp.
type PublicIpStatus struct {
	domain.Status
	// AttachedTo is optional. Nil means unattached.
	AttachedTo *domain.Reference
	IpAddress  string
}
