// Package frontend provides conversions between the regional Reference
// and the SDK schema Reference type, shared across API mapping packages.
package frontend

import (
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// ReferenceToAPI converts a domain.Reference to an sdkschema.Reference.
func ReferenceToAPI(ref domain.Reference) sdkschema.Reference {
	return sdkschema.Reference{
		Provider:  ref.Provider,
		Region:    ref.Region,
		Resource:  ref.Resource,
		Tenant:    ref.Tenant,
		Workspace: ref.Workspace,
	}
}

// ReferencePtrToAPI converts a *domain.Reference to an *sdkschema.Reference.
func ReferencePtrToAPI(ref *domain.Reference) *sdkschema.Reference {
	if ref == nil {
		return nil
	}
	result := ReferenceToAPI(*ref)
	return &result
}

// ReferenceFromAPI converts an sdkschema.Reference to a domain.Reference.
func ReferenceFromAPI(ref sdkschema.Reference) domain.Reference {
	return domain.Reference{
		Provider:  ref.Provider,
		Region:    ref.Region,
		Resource:  ref.Resource,
		Tenant:    ref.Tenant,
		Workspace: ref.Workspace,
	}
}
