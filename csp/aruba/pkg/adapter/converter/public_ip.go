package converter

import (
	"errors"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

// PublicIpElasticIpConverter maps a SECA PublicIp to an Aruba ElasticIP.
//
// An Aruba ElasticIP has neither an address nor an IP-version field: the address is always
// allocated by Aruba and is always IPv4. Specs asking for anything else are rejected rather
// than silently downgraded. See csp/aruba/README.md.
type PublicIpElasticIpConverter struct{}

func NewPublicIpElasticIpConverter() *PublicIpElasticIpConverter {
	return &PublicIpElasticIpConverter{}
}

func (c *PublicIpElasticIpConverter) FromSECAToAruba(from *publicipdom.PublicIp) (*v1alpha1.ElasticIP, error) {
	if from.Spec.Address != "" {
		return nil, kernel.NewError(kernel.KindValidation, errors.New("public ip address cannot be requested: Aruba does not support bring-your-own-IP"))
	}

	if from.Spec.Version == commondomain.IPVersionIPv6 {
		return nil, kernel.NewError(kernel.KindValidation, errors.New("IPv6 public ip is not supported by Aruba"))
	}

	tenant := from.GetTenant()
	workspace := from.GetWorkspace()
	namespace := k8sadapter.ComputeNamespace(from)
	namespaceWorkspace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: tenant})

	region := from.Region
	if region == "" {
		region = defaultRegion
	}

	return &v1alpha1.ElasticIP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      from.Name,
			Namespace: namespace,
			Labels: map[string]string{
				"seca.publicip/workspace":  workspace,
				"seca.publicip/tenant":     tenant,
				"seca.publicip/namespace":  namespace,
				"seca.workspace/namespace": namespaceWorkspace,
			},
		},
		Spec: v1alpha1.ElasticIPSpec{
			Tenant:        tenant,
			Region:        region,
			Tags:          ArubaTags(from.Labels),
			BillingPeriod: defaultBillingPeriod,
			ProjectReference: v1alpha1.ResourceReference{
				Name:      workspace,
				Namespace: namespaceWorkspace,
			},
		},
	}, nil
}

func (c *PublicIpElasticIpConverter) FromArubaToSECA(from *v1alpha1.ElasticIP) (*publicipdom.PublicIp, error) {
	tenant := from.Spec.Tenant
	if tenant == "" {
		tenant = from.Labels["seca.publicip/tenant"]
	}
	if tenant == "" {
		return nil, kernel.NewError(kernel.KindValidation, errors.New("tenant is missing"))
	}

	workspace := from.Spec.ProjectReference.Name
	if workspace == "" {
		workspace = from.Labels["seca.publicip/workspace"]
	}
	if workspace == "" {
		return nil, kernel.NewError(kernel.KindValidation, errors.New("workspace is missing"))
	}

	return &publicipdom.PublicIp{
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
		Spec: publicipdom.PublicIpSpec{
			Version: commondomain.IPVersionIPv4,
		},
	}, nil
}
