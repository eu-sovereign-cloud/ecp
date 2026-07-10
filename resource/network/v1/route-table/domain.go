// Package routetable defines the route table resource domain model and identity constants.
package routetable

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the RouteTable resource.
const (
	Kind       = "RouteTable"
	Resource   = "route-tables"
	Group      = "network.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.network/v1"
)

// RouteTable represents the domain model for a route table.
type RouteTable struct {
	domain.RegionalNetworkMetadata
	Spec   RouteTableSpec
	Status *RouteTableStatus
}

// RouteTableSpec defines the specification for a RouteTable.
type RouteTableSpec struct {
	Routes []RouteSpec
}

// RouteSpec defines a single route within a RouteTable.
type RouteSpec struct {
	// DestinationCidrBlock is the CIDR block matched by this route.
	DestinationCidrBlock string
	// TargetRef references the instance, gateway, or IP address the route forwards to.
	TargetRef domain.Reference
}

// RouteTableStatus defines the status for a RouteTable.
type RouteTableStatus struct {
	domain.Status
	// Routes carries per-route status; left empty by the dummy plugin, which only manages
	// whole-resource state.
	Routes []RouteStatus
}

// RouteStatus defines the status of a single route within a RouteTable.
type RouteStatus struct {
	State      domain.ResourceState
	Conditions []domain.StatusCondition
}
