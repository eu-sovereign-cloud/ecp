package regional

import (
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

type Metadata struct {
	model.CommonMetadata
	scope.Scope

	Labels      map[string]string
	Annotations map[string]string
	Extensions  map[string]string
	Region      string
}
