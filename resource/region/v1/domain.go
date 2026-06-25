// Package v1 defines the region resource domain model and identity constants.
package v1

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the region resource.
const (
	Kind     = "Region"
	Resource = "regions"
	Group    = "v1.secapi.cloud"
	Version  = "v1"

	RegionBaseURL = "/providers/seca.region"
	ProviderID    = "seca.region/v1"
)

// Zone is a string type representing a region zone name.
type Zone string

// Provider represents a region provider.
type Provider struct {
	Name    string
	URL     string
	Version string
}

// Region is the domain model for a region resource.
type Region struct {
	domain.Metadata
	Providers []Provider
	Zones     []Zone
}
