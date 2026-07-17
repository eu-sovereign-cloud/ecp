package rest

import (
	"fmt"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku"
)

const (
	// InstanceSKUAPIVersion is the API version string used in response metadata.
	InstanceSKUAPIVersion = skudom.Version
	// InstanceSKUResource is the resource name used in response metadata.
	InstanceSKUResource = skudom.Resource
)

// instanceSKUToAPIWithVerb returns a func that converts an InstanceSKU to its SDK representation with the given verb.
func instanceSKUToAPIWithVerb(verb string) func(sku *skudom.InstanceSKU) *sdkschema.InstanceSku {
	return func(sku *skudom.InstanceSKU) *sdkschema.InstanceSku {
		sdk := instanceSKUToAPI(sku)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

// instanceSKUToAPI converts an InstanceSKU to its SDK representation.
func instanceSKUToAPI(sku *skudom.InstanceSKU) *sdkschema.InstanceSku {
	return &sdkschema.InstanceSku{
		Metadata: &sdkschema.SkuResourceMetadata{
			ApiVersion: InstanceSKUAPIVersion,
			Kind:       sdkschema.SkuResourceMetadataKindResourceKindInstanceSku,
			Name:       sku.Name,
			Provider:   sku.Provider,
			Region:     sku.Region,
			Tenant:     sku.Tenant,
			Resource:   fmt.Sprintf(commondomain.RegionalResourceFormat, sdkschema.SkuResourceMetadataKindResourceKindInstanceSku, sku.Name),
			Ref: fmt.Sprintf(
				sku.Provider+"/"+commondomain.RegionalTenantScopedResourceFormat,
				sku.Tenant,
				sdkschema.SkuResourceMetadataKindResourceKindInstanceSku,
				sku.Name,
			),
		},
		Spec: &sdkschema.InstanceSkuSpec{
			Ram:  sku.Spec.Ram,
			VCPU: sku.Spec.VCPU,
		},
	}
}

// instanceSKUIteratorToAPI converts a list of InstanceSKU to an SDK compute SkuIterator.
func instanceSKUIteratorToAPI(skus []*skudom.InstanceSKU, nextSkipToken *string) *sdkcompute.SkuIterator {
	items := make([]sdkschema.InstanceSku, len(skus))
	for i := range skus {
		items[i] = *instanceSKUToAPI(skus[i])
	}

	iterator := &sdkcompute.SkuIterator{
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
