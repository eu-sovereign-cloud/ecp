package regional

import "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"

// UpsertParams - parameters for creating or updating resources. Implements port.IdentifiableResource.
type UpsertParams struct {
	scope.Scope

	Name              string
	IfUnmodifiedSince int
}

func (u *UpsertParams) GetName() string { return u.Name }
