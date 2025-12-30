package regional

import "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"

type UpsertParams struct {
	scope.Scope

	Name              string
	IfUnmodifiedSince *int
}

func (u *UpsertParams) GetName() string { return u.Name }
