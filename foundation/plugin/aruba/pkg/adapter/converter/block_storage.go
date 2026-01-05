package converter

import (
	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BlockStorageConverter struct {
}

func (c *BlockStorageConverter) FromSECAToAruba(from *regional.BlockStorageDomain) (*v1alpha1.BlockStorage, error) {

	return &v1alpha1.BlockStorage{
		ObjectMeta: v1.ObjectMeta{
			Name:      from.Metadata.Name,
			Namespace: "default",
		},
		Spec: v1alpha1.BlockStorageSpec{
			SizeGb: int32(from.Spec.SizeGB),
			Tenant: from.Spec.SourceImageRef.Tenant,
			Location: v1alpha1.Location{
				Value: from.Spec.SourceImageRef.Region,
			},
			ProjectReference: v1alpha1.ResourceReference {
				Name: from.Spec.SourceImageRef.Workspace,
			},
				
			DataCenter: "IT-BG1",
			BillingPeriod: "Monthly",

		},
	}, nil

}

func (c *BlockStorageConverter) FromArubaToSECA(from *v1alpha1.BlockStorage) (*regional.BlockStorageDomain, error) {
	return &regional.BlockStorageDomain{
		Metadata: model.Metadata{
			CommonMetadata: model.CommonMetadata{
			Name: from.ObjectMeta.Name,
			},
		},
		Spec: regional.BlockStorageSpec{
			SizeGB: int(from.Spec.SizeGb),
			SkuRef: regional.ReferenceObject{
				
			},
			SourceImageRef: &regional.ReferenceObject{
				Tenant: from.Spec.Tenant,
				Region: from.Spec.Location.Value,
				Workspace: from.Spec.ProjectReference.Name ,
			},
		},
	}, nil
}
