package converter

import (
	"errors"
	"math"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes"
	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"
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

func (c *BlockStorageConverter) FromSECAToAruba(from *bsdom.BlockStorage) (*v1alpha1.BlockStorage, error) {
	tenant := from.GetTenant()
	workspace := from.GetWorkspace()
	namespace := k8sadapter.ComputeNamespace(from) // TODO: ask to change repository for  ComputeNamespace from kubernetes adapter to scope
	namespaceWorkspace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: tenant})
	sizeGB, err := SecaToArubaSize(from.Spec.SizeGB)
	if err != nil {
		return nil, err // TODO: better error handling
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
			SizeGB: sizeGB,
			Tenant: tenant,
			Region: getRegionFromSpecOrDefault(from),
			ProjectReference: v1alpha1.ResourceReference{
				Name:      workspace,
				Namespace: namespaceWorkspace,
			},
			// TODO: must be fixed
			Zone:          defaultDatacenter,
			BillingPeriod: defaultBillingPeriod,
		},
	}, nil
}

func (c *BlockStorageConverter) FromArubaToSECA(from *v1alpha1.BlockStorage) (*bsdom.BlockStorage, error) {
	tenant, err := getTenantFromSpecOrError(from)
	if err != nil {
		return nil, err // TODO: better error handler management
	}
	workspace, err := getWorkspaceFromSpecOrError(from)
	if err != nil {
		return nil, err // TODO: better error handler management
	}

	return &bsdom.BlockStorage{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{
				Name: from.Name,
			},
			Scope: res.Scope{
				Tenant:    tenant,
				Workspace: workspace,
			},
		},
		Spec: bsdom.BlockStorageSpec{
			SizeGB: int(from.Spec.SizeGB),
			SkuRef: commondomain.ReferenceDomain{},
			SourceImageRef: &commondomain.ReferenceDomain{
				Tenant:    from.Spec.Tenant,
				Region:    from.Spec.Region,
				Workspace: from.Spec.ProjectReference.Name,
			},
		},
	}, nil
}

func SecaToArubaSize(in int) (int32, error) {
	if in > math.MaxInt32 || in < math.MinInt32 {
		return 0, errors.New("storage size out of range")
	}

	return int32(in), nil //nolint:gosec // boundaries checked above
}

// getRegionFromSpecOrDefault get region from source image or sku ref otherwise default value
func getRegionFromSpecOrDefault(from *bsdom.BlockStorage) string {
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
