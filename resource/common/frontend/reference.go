// Package frontend provides conversions between the regional ReferenceDomain
// and the SDK schema Reference type, shared across API mapping packages.
package frontend

import (
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// ToAPI converts a domain.ReferenceDomain to an sdkschema.Reference.
func ToAPI(ref domain.ReferenceDomain) sdkschema.Reference {
	return sdkschema.Reference{
		Provider:  ref.Provider,
		Region:    ref.Region,
		Resource:  ref.Resource,
		Tenant:    ref.Tenant,
		Workspace: ref.Workspace,
	}
}

// PtrToAPI converts a *domain.ReferenceDomain to an *sdkschema.Reference.
func PtrToAPI(ref *domain.ReferenceDomain) *sdkschema.Reference {
	if ref == nil {
		return nil
	}
	return new(ToAPI(*ref))
}

// FromAPI converts an sdkschema.Reference to a domain.ReferenceDomain.
func FromAPI(ref sdkschema.Reference) domain.ReferenceDomain {
	return domain.ReferenceDomain{
		Provider:  ref.Provider,
		Region:    ref.Region,
		Resource:  ref.Resource,
		Tenant:    ref.Tenant,
		Workspace: ref.Workspace,
	}
}
