package regional

import "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"

type NetworkSKUDomain struct {
	model.Metadata
	Spec NetworkSKUSpec
}

type NetworkSKUSpec struct {
	Bandwidth int
	Packets   int
}
