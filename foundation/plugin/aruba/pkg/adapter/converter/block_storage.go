package converter

import (
	"errors"
	"log"
	"math"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

type BlockStorageConverter struct {
}

func NewBlockStorageConverter() *BlockStorageConverter {
	return &BlockStorageConverter{}
}

func (c *BlockStorageConverter) FromSECAToAruba(from *regional.BlockStorageDomain) (*v1alpha1.BlockStorage, error) {

	sizeGb, err := secaToArubaSize(from.Spec.SizeGB)
	if err != nil {
		return nil, err //TODO: better error handling
	}

	tenant := from.GetTenant()
	workspace := from.GetWorkspace()

	source := from.Spec.SourceImageRef

	region := ""
	if source != nil {
		region = from.Spec.SourceImageRef.Region
	}

	if region == "" {
		region = from.Spec.SkuRef.Region
	}

	//TODO: ask to change repository for  ComputeNamespace from kubernetes adapter to scope
	namespace := kubernetesadapter.ComputeNamespace(&from.Metadata.Scope)

	namespaceWs := kubernetesadapter.ComputeNamespace(&scope.Scope{
		Tenant: tenant,
	})

	log.Println("project reference", "namespace", namespaceWs, "workspace", workspace)

	bsConv := &v1alpha1.BlockStorage{
		ObjectMeta: v1.ObjectMeta{
			Name:      from.Metadata.Name,
			Namespace: namespace,
			Labels: map[string]string{
				"seca.blockstorage/workspace": workspace,
				"seca.blockstorage/tenant":    tenant,
				"seca.blockstorage/namespace": namespace},
		},
		Spec: v1alpha1.BlockStorageSpec{
			SizeGb: sizeGb,
			Tenant: tenant,
			Location: v1alpha1.Location{
				Value: region,
			},
			ProjectReference: v1alpha1.ResourceReference{
				Name:      workspace,
				Namespace: namespaceWs,
			},

			DataCenter:    "ITBG-1",
			BillingPeriod: "Hour", // supported values: "Hour", "Month",
		},
	}
	log.Println("--->> BS CONVERTED", "result", bsConv)
	return bsConv, nil

}

func (c *BlockStorageConverter) FromArubaToSECA(from *v1alpha1.BlockStorage) (*regional.BlockStorageDomain, error) {
	tenant := from.Labels["seca.blockstorage/tenant"]
	workspace := from.Labels["seca.blockstorage/workspace"]

	return &regional.BlockStorageDomain{
		Metadata: regional.Metadata{
			Scope: scope.Scope{
				Tenant:    tenant,
				Workspace: workspace,
			},
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
