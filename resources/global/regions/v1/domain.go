// Package domain defines the region resource domain model and identity constants.
package v1

import "github.com/eu-sovereign-cloud/ecp/resources/common/domain"

// Identity constants for the region resource.
const (
	Kind     = "Region"
	Resource = "regions"
	Group    = "v1.secapi.cloud"
	Version  = "v1"
)

// Type aliases so callers can use this package without importing common/domain directly.
type (
	Region   = domain.RegionDomain
	Provider = domain.ProviderDomain
	Zone     = domain.ZoneDomain
)
