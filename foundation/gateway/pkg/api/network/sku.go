package network

import (
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
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
