package regional

import (
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// ToSDKStorageSKU converts a StorageSKUDomain to its SDK representation.
func ToSDKStorageSKU(domain *StorageSKUDomain) *sdkschema.StorageSku {
	return &sdkschema.StorageSku{
		Metadata: &sdkschema.SkuResourceMetadata{
			Name: domain.Name, // no namespace?
		},
		Spec: &sdkschema.StorageSkuSpec{
			Iops:          int(domain.Spec.Iops),
			MinVolumeSize: int(domain.Spec.MinVolumeSize),
			Type:          sdkschema.StorageSkuSpecType(domain.Spec.Type),
		},
	}
}
