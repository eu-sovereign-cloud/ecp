package converter

import (
	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BlockStorageConverter struct {
}

func (c *BlockStorageConverter) FromSECAToAruba(from *regional.BlockStorageDomain) (*v1alpha1.BlockStorage, error) {

	return &v1alpha1.BlockStorage{
		ObjectMeta: v1.ObjectMeta{
			Name:      "blockstorage-" + from.Metadata.Name,
			Namespace: "default",
		},
		Spec: v1alpha1.BlockStorageSpec{
			SizeGb: int32(from.Spec.SizeGB),
			Tenant: from.Spec.SourceImageRef.Tenant,
			Location: v1alpha1.Location{
				Value: from.Spec.SourceImageRef.Region,
			},
		},
	}, nil

}

func (c *BlockStorageConverter) FromArubaToSECA(from *v1alpha1.BlockStorage) (*regional.BlockStorageDomain, error) {
	return &regional.BlockStorageDomain{}, nil
}
