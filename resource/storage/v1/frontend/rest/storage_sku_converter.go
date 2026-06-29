package rest

import (
	"fmt"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku"
)

const (
	// StorageSKUAPIVersion is the API version string used in response metadata.
	StorageSKUAPIVersion = skudom.Version
	// StorageSKUResource is the resource name used in response metadata.
	StorageSKUResource = skudom.Resource
)

// StorageSKUToAPIWithVerb returns a func that converts a StorageSKU to its SDK representation with the given verb.
func StorageSKUToAPIWithVerb(verb string) func(sku *skudom.StorageSKU) *sdkschema.StorageSku {
	return func(sku *skudom.StorageSKU) *sdkschema.StorageSku {
		sdk := storageSKUToAPI(sku)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

// storageSKUToAPI converts a StorageSKU to its SDK representation.
func storageSKUToAPI(sku *skudom.StorageSKU) *sdkschema.StorageSku {
	return &sdkschema.StorageSku{
		Metadata: &sdkschema.SkuResourceMetadata{
			ApiVersion: StorageSKUAPIVersion,
			Kind:       sdkschema.SkuResourceMetadataKindResourceKindStorageSku,
			Name:       sku.Name,
			Provider:   sku.Provider,
			Region:     sku.Region,
			Tenant:     sku.Tenant,
			Resource:   fmt.Sprintf(commondomain.RegionalResourceFormat, sdkschema.SkuResourceMetadataKindResourceKindStorageSku, sku.Name),
			Ref: fmt.Sprintf(
				sku.Provider+"/"+commondomain.RegionalTenantScopedResourceFormat,
				sku.Tenant,
				sdkschema.SkuResourceMetadataKindResourceKindStorageSku,
				sku.Name,
			),
		},
		Spec: &sdkschema.StorageSkuSpec{
			Iops:          int(sku.Spec.IOPS),
			MinVolumeSize: int(sku.Spec.MinVolumeSize),
			Type:          sdkschema.StorageSkuSpecType(sku.Spec.Type),
		},
	}
}

// StorageSKUIteratorToAPI converts a list of StorageSKU to an SDK SkuIterator.
func StorageSKUIteratorToAPI(skus []*skudom.StorageSKU, nextSkipToken *string) *sdkstorage.SkuIterator {
	items := make([]sdkschema.StorageSku, len(skus))
	for i := range skus {
		items[i] = *storageSKUToAPI(skus[i])
	}

	iterator := &sdkstorage.SkuIterator{
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
