package network

import (
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

const (
	BaseURL             = "/providers/seca.network"
	ProviderStorageName = "seca.network/v1"
)

// DomainToAPI converts a NetworkSKUDomain to its API representation.
func DomainToAPI(domain *regional.NetworkSKUDomain) *sdkschema.NetworkSku {
	return &sdkschema.NetworkSku{
		Metadata: &sdkschema.SkuResourceMetadata{Name: domain.Name},
		Spec: &sdkschema.NetworkSkuSpec{
			Bandwidth: domain.Spec.Bandwidth,
			Packets:   domain.Spec.Packets,
		},
	}
}
