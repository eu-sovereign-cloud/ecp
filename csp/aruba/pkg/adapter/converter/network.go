package converter

import (
	"errors"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

// NetworkVPCConverter maps a SECA Network to an Aruba VPC.
//
// The SECA CIDR/AdditionalCIDRs and SkuRef are not propagated: an Aruba VPC has no address
// range of its own (addressing is defined per subnet) and no SKU concept. See csp/aruba/README.md.
type NetworkVPCConverter struct{}

func NewNetworkVPCConverter() *NetworkVPCConverter {
	return &NetworkVPCConverter{}
}

func (c *NetworkVPCConverter) FromSECAToAruba(from *netdom.Network) (*v1alpha1.VPC, error) {
	tenant := from.GetTenant()
	workspace := from.GetWorkspace()
	namespace := k8sadapter.ComputeNamespace(from)
	namespaceWorkspace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: tenant})

	region := from.Region
	if region == "" {
		region = defaultRegion
	}

	return &v1alpha1.VPC{
		ObjectMeta: metav1.ObjectMeta{
			Name:      from.Name,
			Namespace: namespace,
			Labels: map[string]string{
				"seca.network/workspace":   workspace,
				"seca.network/tenant":      tenant,
				"seca.network/namespace":   namespace,
				"seca.workspace/namespace": namespaceWorkspace,
			},
		},
		Spec: v1alpha1.VPCSpec{
			Tenant: tenant,
			Region: region,
			ProjectReference: v1alpha1.ResourceReference{
				Name:      workspace,
				Namespace: namespaceWorkspace,
			},
		},
	}, nil
}

func (c *NetworkVPCConverter) FromArubaToSECA(from *v1alpha1.VPC) (*netdom.Network, error) {
	tenant := from.Spec.Tenant
	if tenant == "" {
		tenant = from.Labels["seca.network/tenant"]
	}
	if tenant == "" {
		return nil, errors.New("tenant is missing")
	}

	workspace := from.Spec.ProjectReference.Name
	if workspace == "" {
		workspace = from.Labels["seca.network/workspace"]
	}
	if workspace == "" {
		return nil, errors.New("workspace is missing")
	}

	return &netdom.Network{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{
				Name: from.Name,
			},
			Scope: res.Scope{
				Tenant:    tenant,
				Workspace: workspace,
			},
			Region: from.Spec.Region,
		},
	}, nil
}
