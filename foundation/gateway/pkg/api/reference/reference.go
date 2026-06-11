// Package reference provides conversions between the regional ReferenceDomain
// and the SDK schema Reference type, shared across API mapping packages.
package reference

import (
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
)

// ToAPI converts a regional.ReferenceDomain to an sdkschema.Reference.
func ToAPI(ref regional.ReferenceDomain) sdkschema.Reference {
	return sdkschema.Reference{
		Provider:  ref.Provider,
		Region:    ref.Region,
		Resource:  ref.Resource,
		Tenant:    ref.Tenant,
		Workspace: ref.Workspace,
	}
}

// PtrToAPI converts a *regional.ReferenceDomain to an *sdkschema.Reference.
func PtrToAPI(ref *regional.ReferenceDomain) *sdkschema.Reference {
	if ref == nil {
		return nil
	}
	return new(ToAPI(*ref))
}

// FromAPI converts an sdkschema.Reference to a regional.ReferenceDomain.
func FromAPI(ref sdkschema.Reference) regional.ReferenceDomain {
	return regional.ReferenceDomain{
		Provider:  ref.Provider,
		Region:    ref.Region,
		Resource:  ref.Resource,
		Tenant:    ref.Tenant,
		Workspace: ref.Workspace,
	}
}
