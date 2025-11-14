package regional

import (
	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/block-storage/skus/v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

func FromUnstructuredToStorageSKUDomain(u unstructured.Unstructured) (*StorageSKUDomain, error) {
	var crdStorageSKU skuv1.StorageSKU
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &crdStorageSKU); err != nil {
		return nil, err
	}
	return FromCRToStorageSKUDomain(crdStorageSKU), nil
}

func FromCRToStorageSKUDomain(cr skuv1.StorageSKU) *StorageSKUDomain {
	return &StorageSKUDomain{
		Meta: model.Metadata{
			Name: cr.Name,
		},
		Spec: StorageSKUSpec{
			Iops:          int64(cr.Spec.Iops),
			MinVolumeSize: int64(cr.Spec.MinVolumeSize),
			Type:          string(cr.Spec.Type),
		},
	}
}

func ToSDKStorageSKU(domain *StorageSKUDomain) *sdkschema.StorageSku {
	return &sdkschema.StorageSku{
		Metadata: &sdkschema.SkuResourceMetadata{
			Name: domain.Meta.Name, // no namespace?
		},
		Spec: &sdkschema.StorageSkuSpec{
			Iops:          int(domain.Spec.Iops),
			MinVolumeSize: int(domain.Spec.MinVolumeSize),
			Type:          sdkschema.StorageSkuSpecType(domain.Spec.Type),
		},
	}
}
