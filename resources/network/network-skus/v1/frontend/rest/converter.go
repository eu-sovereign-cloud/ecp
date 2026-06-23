// Package rest provides REST↔domain conversion functions for the network SKU resource.
package rest

import (
	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	skudom "github.com/eu-sovereign-cloud/ecp/resources/network/network-skus/v1"
)

// NetworkSKUDomainToAPI converts a NetworkSKU to its SDK representation.
func NetworkSKUDomainToAPI(domain *skudom.NetworkSKU) *sdkschema.NetworkSku {
	return &sdkschema.NetworkSku{
		Metadata: &sdkschema.SkuResourceMetadata{Name: domain.Name},
		Spec: &sdkschema.NetworkSkuSpec{
			Bandwidth: domain.Spec.Bandwidth,
			Packets:   domain.Spec.Packets,
		},
	}
}

// NetworkSKUDomainToAPIIterator converts a list of NetworkSKU to an SDK SkuIterator.
func NetworkSKUDomainToAPIIterator(domains []*skudom.NetworkSKU, nextSkipToken *string) *sdknetwork.SkuIterator {
	items := make([]sdkschema.NetworkSku, len(domains))
	for i := range domains {
		items[i] = *NetworkSKUDomainToAPI(domains[i])
	}

	iterator := &sdknetwork.SkuIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: skudom.ProviderID,
			Resource: skudom.Resource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}
