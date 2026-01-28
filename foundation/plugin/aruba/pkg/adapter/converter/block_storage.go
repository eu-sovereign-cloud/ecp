package converter

import (
	"errors"
	"math"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BlockStorageConverter struct {
	namespace string
}

func NewBlockStorageConverter(namespace string) *BlockStorageConverter {
	return &BlockStorageConverter{
		namespace: namespace,
	}
}

func (c *BlockStorageConverter) FromSECAToAruba(from *regional.BlockStorageDomain) (*v1alpha1.BlockStorage, error) {

	sizeGb, err := secaToArubaSize(from.Spec.SizeGB)
	if err != nil {
		return nil, err //TODO: better error handling
	}

	tenant := from.Metadata.Scope.Tenant
	region := from.Metadata.Region

	workspace := ""
	if from.Metadata.Labels != nil {
		workspace = from.Metadata.Labels["secapi.cloud/workspace"]
	}

	return &v1alpha1.BlockStorage{
		ObjectMeta: v1.ObjectMeta{
			Name:      from.Metadata.Name,
			Namespace: c.namespace,
			Labels: map[string]string{
				"seca.blockstorage/id": from.Metadata.Name,
			},
		},
		Spec: v1alpha1.BlockStorageSpec{
			SizeGb: sizeGb,
			Tenant: tenant,
			Location: v1alpha1.Location{
				Value: region,
			},
			ProjectReference: v1alpha1.ResourceReference{
				Name:      workspace,
				Namespace: c.namespace,
			},

			DataCenter:    "IT-BG1",
			BillingPeriod: "Hour", // supported values: "Hour", "Month",
		},
	}, nil

}

func (c *BlockStorageConverter) FromArubaToSECA(from *v1alpha1.BlockStorage) (*regional.BlockStorageDomain, error) {
	return &regional.BlockStorageDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: from.Name,
			},
		},
		Spec: regional.BlockStorageSpec{
			SizeGB: int(from.Spec.SizeGb),
			SkuRef: regional.ReferenceObject{},
			SourceImageRef: &regional.ReferenceObject{
				Tenant:    from.Spec.Tenant,
				Region:    from.Spec.Location.Value,
				Workspace: from.Spec.ProjectReference.Name,
			},
		},
	}, nil
}

func secaToArubaSize(in int) (int32, error) {
	if in > math.MaxInt32 || in < math.MinInt32 {
		return 0, errors.New("storage size out of range")
	}

	return int32(in), nil //nolint:gosec // boundaries checked above
}
