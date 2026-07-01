package rest

import (
	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	skudom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"
)

// networkSKUToAPI converts a NetworkSKU to its SDK representation.
func networkSKUToAPI(sku *skudom.NetworkSKU) *sdkschema.NetworkSku {
	return &sdkschema.NetworkSku{
		Metadata: &sdkschema.SkuResourceMetadata{Name: sku.Name},
		Spec: &sdkschema.NetworkSkuSpec{
			Bandwidth: sku.Spec.Bandwidth,
			Packets:   sku.Spec.Packets,
		},
	}
}

// networkSKUIteratorToAPI converts a list of NetworkSKU to an SDK SkuIterator.
func networkSKUIteratorToAPI(skus []*skudom.NetworkSKU, nextSkipToken *string) *sdknetwork.SkuIterator {
	items := make([]sdkschema.NetworkSku, len(skus))
	for i := range skus {
		items[i] = *networkSKUToAPI(skus[i])
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
