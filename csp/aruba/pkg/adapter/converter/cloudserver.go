package converter

import (
	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// CloudServerRefs holds the Aruba references the compute-instance handler resolves before a
// CloudServer can be assembled: a SECA Instance names none of them directly (they come from its
// NICs, boot/data volumes, ssh keys and sku), so they are gathered by the handler and passed here.
type CloudServerRefs struct {
	FlavorName string
	// Zone is the boot volume's zone: Aruba requires a CloudServer and its boot volume to share one,
	// and the volume's is the zone that actually exists (SECA models no per-volume zone).
	Zone                    string
	VPCReference            v1alpha1.ResourceReference
	SubnetReferences        []v1alpha1.ResourceReference
	SecurityGroupReferences []v1alpha1.ResourceReference
	KeyPairReference        v1alpha1.ResourceReference
	BootVolumeReference     v1alpha1.ResourceReference
	DataVolumeReferences    []v1alpha1.ResourceReference
	ElasticIPReference      *v1alpha1.ResourceReference
	ProjectReference        v1alpha1.ResourceReference
}

// BuildCloudServer maps a SECA Instance plus its resolved Aruba references to an Aruba CloudServer,
// living in the same workspace-level namespace as the VPC and block storages it references.
func BuildCloudServer(from *instancedom.Instance, refs CloudServerRefs) *v1alpha1.CloudServer {
	tenant := from.GetTenant()
	workspace := from.GetWorkspace()
	namespace := k8sadapter.ComputeNamespace(from)

	region := from.Region
	if region == "" {
		region = defaultRegion
	}
	zone := refs.Zone
	if zone == "" {
		zone = defaultDatacenter
	}

	return &v1alpha1.CloudServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      from.Name,
			Namespace: namespace,
			Labels: map[string]string{
				"seca.instance/workspace": workspace,
				"seca.instance/tenant":    tenant,
				"seca.instance/namespace": namespace,
			},
		},
		Spec: v1alpha1.CloudServerSpec{
			Tenant:                  tenant,
			Region:                  region,
			Tags:                    ArubaTags(from.Labels),
			Zone:                    zone,
			FlavorName:              refs.FlavorName,
			VPCReference:            refs.VPCReference,
			SubnetReferences:        refs.SubnetReferences,
			SecurityGroupReferences: refs.SecurityGroupReferences,
			KeyPairReference:        refs.KeyPairReference,
			BootVolumeReference:     refs.BootVolumeReference,
			DataVolumeReferences:    refs.DataVolumeReferences,
			ElasticIPReference:      refs.ElasticIPReference,
			ProjectReference:        refs.ProjectReference,
		},
	}
}
