package converter

import (
	"errors"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

const (
	// subnetTypeAdvanced lets the subnet use the CIDR carried by the SECA spec. "Basic" would
	// have Aruba pick the address range itself, which would silently ignore that CIDR.
	subnetTypeAdvanced = "Advanced"
)

// Labels stamped on every Aruba Subnet. A SECA subnet is network-scoped, so the same name may
// exist in several networks of one workspace and these are what the compute-instance handler
// searches on to find the right one - keep the writer below and that reader in step.
const (
	LabelSubnetTenant    = "seca.subnet/tenant"
	LabelSubnetWorkspace = "seca.subnet/workspace"
	LabelSubnetNetwork   = "seca.subnet/network"
)

// SubnetConverter maps a SECA Subnet to an Aruba Subnet.
//
// The SECA Zone, SkuRef and RouteTableRef are not propagated: the Aruba Subnet CRD has no
// equivalent fields. DHCP is always enabled because the CRD requires the field and SECA has no
// knob for it. See csp/aruba/README.md.
type SubnetConverter struct{}

func NewSubnetConverter() *SubnetConverter {
	return &SubnetConverter{}
}

func (c *SubnetConverter) FromSECAToAruba(from *subnetdom.Subnet) (*v1alpha1.Subnet, error) {
	// Aruba subnets are IPv4-only (the CRD's CIDR field is validated against a dotted-quad
	// pattern), so an IPv6-only SECA subnet has nothing to map onto.
	if from.Spec.Cidr.IPv4 == "" {
		return nil, errors.New("subnet requires an IPv4 CIDR: Aruba does not support IPv6-only subnets")
	}

	tenant := from.GetTenant()
	workspace := from.GetWorkspace()
	network := from.GetNetwork()
	namespace := k8sadapter.ComputeNetworkNamespace(from)
	namespaceWorkspace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: tenant})
	// The VPC is created by NetworkVPCConverter in the workspace-level namespace, named after
	// the SECA network this subnet is scoped under.
	namespaceVPC := k8sadapter.ComputeNamespace(&res.Scope{Tenant: tenant, Workspace: workspace})

	region := from.Region
	if region == "" {
		region = defaultRegion
	}

	return &v1alpha1.Subnet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      from.Name,
			Namespace: namespace,
			Labels: map[string]string{
				LabelSubnetWorkspace:       workspace,
				LabelSubnetTenant:          tenant,
				LabelSubnetNetwork:         network,
				"seca.subnet/namespace":    namespace,
				"seca.workspace/namespace": namespaceWorkspace,
			},
		},
		Spec: v1alpha1.SubnetSpec{
			Tenant: tenant,
			Region: region,
			Tags:   ArubaTags(from.Labels),
			Type:   subnetTypeAdvanced,
			CIDR:   from.Spec.Cidr.IPv4,
			DHCP:   v1alpha1.SubnetDHCP{Enabled: true},
			VPCReference: v1alpha1.ResourceReference{
				Name:      network,
				Namespace: namespaceVPC,
			},
			ProjectReference: v1alpha1.ResourceReference{
				Name:      workspace,
				Namespace: namespaceWorkspace,
			},
		},
	}, nil
}

func (c *SubnetConverter) FromArubaToSECA(from *v1alpha1.Subnet) (*subnetdom.Subnet, error) {
	tenant := from.Spec.Tenant
	if tenant == "" {
		tenant = from.Labels["seca.subnet/tenant"]
	}
	if tenant == "" {
		return nil, errors.New("tenant is missing")
	}

	workspace := from.Spec.ProjectReference.Name
	if workspace == "" {
		workspace = from.Labels["seca.subnet/workspace"]
	}
	if workspace == "" {
		return nil, errors.New("workspace is missing")
	}

	network := from.Spec.VPCReference.Name
	if network == "" {
		network = from.Labels["seca.subnet/network"]
	}
	if network == "" {
		return nil, errors.New("network is missing")
	}

	return &subnetdom.Subnet{
		RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
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
			Network: network,
		},
		Spec: subnetdom.SubnetSpec{
			Cidr: subnetdom.CIDR{IPv4: from.Spec.CIDR},
		},
	}, nil
}
