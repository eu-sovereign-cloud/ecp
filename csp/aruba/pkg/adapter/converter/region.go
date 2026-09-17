package converter

import (
	"errors"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
)

// RequireRegion refuses a SECA resource that carries no region.
//
// Every Aruba resource is created in a region, and the only place that region may come from is the
// SECA resource itself (`RegionalMetadata.Region`, stamped on the CR by the regional gateway).
// Defaulting a missing one would provision into a region nobody asked for and hide the upstream bug
// that lost it, so the reconcile fails here instead.
//
// Sending the region empty is not an option either: Aruba resolves both a zone and the size catalog
// *within* a region, so it rejects a resource with none as
// "Validation: Size: invalid; DataCenter: invalid" - an error that names neither the region nor the
// real problem.
func RequireRegion(region string) error {
	if region == "" {
		return kernel.NewError(kernel.KindValidation, errors.New("region is missing: it must be set on the SECA resource"))
	}

	return nil
}
