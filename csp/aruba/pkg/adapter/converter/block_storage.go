package converter

import (
	"errors"
	"math"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

const (
	defaultRegion = "ITBG-Bergamo"
	// defaultDatacenter is the zone every block storage lands in (SECA models no per-volume zone).
	// An Instance carries its own zone, and Aruba requires a CloudServer and its boot volume to share
	// one, so an instance in another zone cannot be satisfied today.
	// ponytail: single hardcoded zone; thread the zone through from the instance if that is needed.
	defaultDatacenter    = "ITBG-1"
	defaultBillingPeriod = "Hour" // supported values: "Hour", "Month"
)

type BlockStorageConverter struct{}

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
			Tags:   ArubaTags(from.Labels),
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
			SkuRef: commondomain.Reference{},
			SourceImageRef: &commondomain.Reference{
				Tenant:    from.Spec.Tenant,
				Region:    from.Spec.Region,
				Workspace: from.Spec.ProjectReference.Name,
			},
		},
	}, nil
}

func SecaToArubaSize(in int) (int32, error) {
	if in > math.MaxInt32 || in < math.MinInt32 {
		return 0, kernel.NewError(kernel.KindValidation, errors.New("storage size out of range"))
	}

	return int32(in), nil //nolint:gosec // boundaries checked above
}

// getRegionFromSpecOrDefault get region from source image or sku ref otherwise default value.
// A SECA reference only carries a region when it points at another one; the common case leaves it
// empty ("inferred from context"), so an empty region must fall through to the default rather than
// be forwarded. Aruba rejects a volume with no region as "Size: invalid; DataCenter: invalid" - both
// a zone and the size catalog are resolved within a region - which is a confusing way to be told the
// location is missing.
func getRegionFromSpecOrDefault(from *bsdom.BlockStorage) string {
	if from.Spec.SourceImageRef != nil && from.Spec.SourceImageRef.Region != "" {
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

	return "", kernel.NewError(kernel.KindValidation, errors.New("tenant is missing"))
}

// getWorkspaceFromSpecOrLabels
func getWorkspaceFromSpecOrError(from *v1alpha1.BlockStorage) (string, error) {
	if from.Spec.ProjectReference.Name != "" {
		return from.Spec.ProjectReference.Name, nil
	}

	if from.Labels["seca.blockstorage/workspace"] != "" {
		return from.Labels["seca.blockstorage/workspace"], nil
	}

	return "", kernel.NewError(kernel.KindValidation, errors.New("workspace is missing"))
}
