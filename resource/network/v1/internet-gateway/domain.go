// Package internetgateway defines the internet gateway resource domain model and identity constants.
package internetgateway

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the InternetGateway resource.
const (
	Kind       = "InternetGateway"
	Resource   = "internet-gateways"
	Group      = "network.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.network/v1"
)

// InternetGateway represents the domain model for an internet gateway.
type InternetGateway struct {
	domain.RegionalMetadata
	Spec   InternetGatewaySpec
	Status *InternetGatewayStatus
}

// InternetGatewaySpec defines the specification for an InternetGateway.
type InternetGatewaySpec struct {
	// EgressOnly, if true, restricts the gateway to outgoing traffic only. Defaults to false.
	EgressOnly bool
}

// InternetGatewayStatus defines the status for an InternetGateway. It carries no
// resource-specific fields — mirrors network.NetworkStatus.
type InternetGatewayStatus struct {
	domain.Status
}
