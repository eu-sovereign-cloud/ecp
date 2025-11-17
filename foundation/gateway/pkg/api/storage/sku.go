package storage

import (
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

// SkuToApi converts a StorageSKUDomain to its SDK representation.
func SkuToApi(domain *regional.StorageSKUDomain) *sdkschema.StorageSku {
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
