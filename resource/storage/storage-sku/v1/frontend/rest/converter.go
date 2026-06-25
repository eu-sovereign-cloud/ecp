// Package rest provides REST↔domain conversion and HTTP handlers for the storage SKU resource.
package rest

import (
	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	skudom "github.com/eu-sovereign-cloud/ecp/resource/storage/storage-sku/v1"
)

// StorageSKUToAPI converts a StorageSKU to its SDK representation.
func StorageSKUToAPI(sku *skudom.StorageSKU) *sdkschema.StorageSku {
	return &sdkschema.StorageSku{
		Metadata: &sdkschema.SkuResourceMetadata{
			Name: sku.Name,
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
		items[i] = *StorageSKUToAPI(skus[i])
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
