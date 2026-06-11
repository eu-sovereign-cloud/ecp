package domain

import "github.com/eu-sovereign-cloud/ecp/foundation/models/scope"

// ListParams - parameters for listing resources
type ListParams struct {
	scope.Scope

	Limit     int
	SkipToken string
	Selector  string
}
