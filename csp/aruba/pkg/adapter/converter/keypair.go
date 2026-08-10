package converter

import (
	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// KeyPairSuffix is appended to the instance name to name the KeyPair it owns. An Aruba
// CloudServer requires a KeyPairReference, but SECA has no key-pair resource: the public key
// travels inline in Instance.Spec.SshKeys. The compute-instance handler therefore materialises a
// KeyPair per instance from the first ssh key and deletes it with the instance.
const KeyPairSuffix = "-key"

// BuildKeyPair maps a SECA Instance and one of its ssh public keys to an Aruba KeyPair, living in
// the same (workspace-level) namespace as the CloudServer it belongs to.
//
// SECA allows up to 32 ssh keys while an Aruba KeyPair carries a single value, so the caller
// passes the key to use (the first). An instance with no ssh key cannot satisfy the CloudServer's
// required KeyPairReference and is gated by the handler rather than reaching this function.
func BuildKeyPair(from *instancedom.Instance, sshKey string) *v1alpha1.KeyPair {
	tenant := from.GetTenant()
	workspace := from.GetWorkspace()
	namespace := k8sadapter.ComputeNamespace(from)
	namespaceWorkspace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: tenant})

	region := from.Region
	if region == "" {
		region = defaultRegion
	}

	return &v1alpha1.KeyPair{
		ObjectMeta: metav1.ObjectMeta{
			Name:      from.Name + KeyPairSuffix,
			Namespace: namespace,
			Labels: map[string]string{
				"seca.instance/workspace":  workspace,
				"seca.instance/tenant":     tenant,
				"seca.instance/instance":   from.Name,
				"seca.instance/namespace":  namespace,
				"seca.workspace/namespace": namespaceWorkspace,
			},
		},
		Spec: v1alpha1.KeyPairSpec{
			Tenant: tenant,
			Region: region,
			// The KeyPair has no SECA resource of its own; it inherits the owning instance's labels.
			Tags:  ArubaTags(from.Labels),
			Value: sshKey,
			ProjectReference: v1alpha1.ResourceReference{
				Name:      workspace,
				Namespace: namespaceWorkspace,
			},
		},
	}
}
