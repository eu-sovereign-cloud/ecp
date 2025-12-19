package model

import "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"

// ListParams - parameters for listing resources
type ListParams struct {
	scope.Scope

	Limit     int
	SkipToken string
	Selector  string
}
