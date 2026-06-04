package regional

import (
	model "github.com/eu-sovereign-cloud/ecp/foundation/models/domain"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/scope"
)

type Metadata struct {
	model.CommonMetadata
	scope.Scope

	Labels      map[string]string
	Annotations map[string]string
	Extensions  map[string]string
	Region      string
}
