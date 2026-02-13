package compute

import (
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

const (
	BaseURL             = "/providers/seca.compute"
	ProviderStorageName = "seca.compute/v1"
)

// SkuToAPI converts a NetworkSKUDomain to its SDK representation.
func SkuToAPI(domain *regional.NetworkSKUDomain) *sdkschema.NetworkSku {
	return &sdkschema.NetworkSku{
		Metadata: &sdkschema.SkuResourceMetadata{Name: domain.Name},
		Spec: &sdkschema.NetworkSkuSpec{
			Bandwidth: domain.Spec.Bandwidth,
			Packets:   domain.Spec.Packets,
		},
	}
}
