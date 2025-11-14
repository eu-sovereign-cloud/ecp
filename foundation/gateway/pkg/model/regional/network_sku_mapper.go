package regional

import (
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// ToSDKNetworkSKU converts a NetworkSKUDomain to its SDK representation.
func ToSDKNetworkSKU(domain *NetworkSKUDomain) *sdkschema.NetworkSku {
	return &sdkschema.NetworkSku{
		Metadata: &sdkschema.SkuResourceMetadata{Name: domain.Name},
		Spec: &sdkschema.NetworkSkuSpec{
			Bandwidth: domain.Spec.Bandwidth,
			Packets:   domain.Spec.Packets,
		},
	}
}
