// Package instancesku defines the compute instance SKU resource domain model and identity constants.
package instancesku

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the instance SKU resource.
const (
	Kind       = "InstanceSKU"
	Resource   = "skus"
	Group      = "compute.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.compute/v1"
)

// InstanceSKU represents the domain model for a compute instance SKU.
type InstanceSKU struct {
	domain.RegionalMetadata
	Spec InstanceSKUSpec
}

// InstanceSKUSpec defines the specification for a compute instance SKU.
type InstanceSKUSpec struct {
	VCPU int
	Ram  int
}
