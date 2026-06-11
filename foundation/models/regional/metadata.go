package regional

import (
	model "github.com/eu-sovereign-cloud/ecp/foundation/models"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/scope"
)

type Metadata struct {
	model.CommonMetadata
	scope.Scope

	Labels      map[string]string
	Annotations map[string]string
	Extensions  map[string]string
	Region      string
}
