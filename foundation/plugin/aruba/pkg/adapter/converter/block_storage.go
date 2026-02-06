package converter

import (
	"errors"
	"math"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

const (
	defaultRegion        = "ITBG-Bergamo"
	defaultDatacenter    = "ITBG-1"
	defaultBillingPeriod = "Hour" // supported values: "Hour", "Month"
)

type BlockStorageConverter struct {
}

func NewBlockStorageConverter() *BlockStorageConverter {
	return &BlockStorageConverter{}
}

func (c *BlockStorageConverter) FromSECAToAruba(from *regional.BlockStorageDomain) (*v1alpha1.BlockStorage, error) {
	tenant := from.GetTenant()
	workspace := from.GetWorkspace()
	namespace := kubernetesadapter.ComputeNamespace(from) //TODO: ask to change repository for  ComputeNamespace from kubernetes adapter to scope
	namespaceWorkspace := kubernetesadapter.ComputeNamespace(&scope.Scope{Tenant: tenant})
	sizeGb, err := secaToArubaSize(from.Spec.SizeGB)
	if err != nil {
		return nil, err //TODO: better error handling
	}

	return &v1alpha1.BlockStorage{
		ObjectMeta: v1.ObjectMeta{
			Name:      from.Name,
			Namespace: namespace,
			Labels: map[string]string{
				"seca.blockstorage/workspace": workspace,
				"seca.blockstorage/tenant":    tenant,
				"seca.blockstorage/namespace": namespace,
				"seca.workspace/namespace":    namespaceWorkspace,
			},
		},
		Spec: v1alpha1.BlockStorageSpec{
			SizeGb: sizeGb,
			Tenant: tenant,
			Location: v1alpha1.Location{
				Value: getRegionFromSpecOrDefault(from),
			},
			ProjectReference: v1alpha1.ResourceReference{
				Name:      workspace,
				Namespace: namespaceWorkspace,
			},
			//todo: must be fixed
			DataCenter:    defaultDatacenter,
			BillingPeriod: defaultBillingPeriod,
		},
	}, nil
}

func (c *BlockStorageConverter) FromArubaToSECA(from *v1alpha1.BlockStorage) (*regional.BlockStorageDomain, error) {
	tenant, err := getTenantFromSpecOrError(from)
	if err != nil {
		return nil, err //TODO: better error handler management
	}
	workspace, err := getWorkspaceFromSpecOrError(from)
	if err != nil {
		return nil, err //TODO: better error handler management
	}

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

// getRegionFromSpecOrDefault get region from source image or sku ref otherwise default value
func getRegionFromSpecOrDefault(from *regional.BlockStorageDomain) string {
	if from.Spec.SourceImageRef != nil {
		return from.Spec.SourceImageRef.Region
	}

	if from.Spec.SkuRef.Region != "" {
		return from.Spec.SkuRef.Region
	}

	return defaultRegion
}

// getTenantFromSpecOrLabel find on spec
func getTenantFromSpecOrError(from *v1alpha1.BlockStorage) (string, error) {
	if from.Spec.Tenant != "" {
		return from.Spec.Tenant, nil
	}

	if from.Labels["seca.blockstorage/tenant"] != "" {
		return from.Labels["seca.blockstorage/tenant"], nil
	}

	return "", errors.New("tenant is missing")
}

// getWorkspaceFromSpecOrLabels
func getWorkspaceFromSpecOrError(from *v1alpha1.BlockStorage) (string, error) {
	if from.Spec.ProjectReference.Name != "" {
		return from.Spec.ProjectReference.Name, nil
	}

	if from.Labels["seca.blockstorage/workspace"] != "" {
		return from.Labels["seca.blockstorage/workspace"], nil
	}

	return "", errors.New("workspace is missing")
}
