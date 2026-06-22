// Package rest provides REST↔domain conversion functions for the storage SKU resource.
package rest

import (
	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	skudom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/storage-skus/v1"
)

// StorageSKUDomainToAPI converts a StorageSKU to its SDK representation.
func StorageSKUDomainToAPI(domain *skudom.StorageSKU) *sdkschema.StorageSku {
	return &sdkschema.StorageSku{
		Metadata: &sdkschema.SkuResourceMetadata{
			Name: domain.Name,
		},
		Spec: &sdkschema.StorageSkuSpec{
			Iops:          int(domain.Spec.Iops),
			MinVolumeSize: int(domain.Spec.MinVolumeSize),
			Type:          sdkschema.StorageSkuSpecType(domain.Spec.Type),
		},
	}
}

// StorageSKUDomainToAPIIterator converts a list of StorageSKU to an SDK SkuIterator.
func StorageSKUDomainToAPIIterator(domains []*skudom.StorageSKU, nextSkipToken *string) *sdkstorage.SkuIterator {
	items := make([]sdkschema.StorageSku, len(domains))
	for i := range domains {
		items[i] = *StorageSKUDomainToAPI(domains[i])
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
