package plugin

import (
	"context"
	"log/slog"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	v2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/adapter/crossplane"
	backend "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	kresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes"
	netdom "github.com/eu-sovereign-cloud/ecp/resources/regional/network/networks/v1"
)

// Network handles create/delete of SECA Network resources on IONOS Cloud.
//
// Mapping:
//   - SECA Network → IONOS private Lan (inside the workspace Datacenter)
//   - IPv6 CIDR (/64) → Lan.IPv6CidrBlock (passthrough)
//   - IPv4 CIDR → no IONOS equivalent for private ranges (RFC 1918); the CIDR
//     is stored in the SECA Network CR and enforced at Subnet/NIC creation time.
type Network struct {
	client client.Client
	logger *slog.Logger
}

func NewNetwork(client client.Client, logger *slog.Logger) *Network {
	return &Network{client: client, logger: logger}
}

func (n *Network) Create(ctx context.Context, resource *netdom.NetworkDomain) error {
	n.logger.Info("ionos network plugin: Create called", "resource_name", resource.GetName())

	namespace := k8sadapter.ComputeNamespace(&kresource.Scope{Tenant: resource.GetTenant()})
	name := resource.GetName()

	// Idempotency: if the Lan already exists, nothing to do.
	existing := &ionosv1alpha1.Lan{}
	if err := n.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, existing); err == nil {
		n.logger.Info("ionos lan already exists, skipping create", "namespace", namespace, "lan", name)
		return nil
	} else if !apierrors.IsNotFound(err) {
		n.logger.Error("failed to check lan existence", "namespace", namespace, "lan", name, "error", err)
		return err
	}

	lan := &ionosv1alpha1.Lan{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ionosv1alpha1.CRDGroupVersion.String(),
			Kind:       ionosv1alpha1.Lan_Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: ionosv1alpha1.LanSpec{
			ManagedResourceSpec: v2.ManagedResourceSpec{
				ProviderConfigReference: &v1.ProviderConfigReference{
					Name: crossplane.ProviderConfigName,
					Kind: crossplane.ProviderConfigType,
				},
			},
			ForProvider: ionosv1alpha1.LanParameters{
				Name: new(name),
				DatacenterIDRef: &v1.NamespacedReference{
					Name:      resource.GetWorkspace(),
					Namespace: namespace,
				},
				Public: new(false),
			},
		},
	}

	// IPv6 CIDRs (/64) are natively supported on IONOS Lan.
	// TODO: additionalCidrs IPv6 — a single Lan has one IPv6CidrBlock; multiple
	// IPv6 ranges would require extra LANs (open question #1).
	if resource.Spec.Cidr.IPv6 != "" {
		lan.Spec.ForProvider.IPv6CidrBlock = new(resource.Spec.Cidr.IPv6)
	}

	if err := n.client.Create(ctx, lan); err != nil {
		n.logger.Error("failed to create lan", "namespace", namespace, "lan", name, "error", err)
		return err
	}

	n.logger.Info("lan created successfully", "namespace", namespace, "lan", name)
	return nil
}

func (n *Network) Delete(ctx context.Context, resource *netdom.NetworkDomain) error {
	n.logger.Info("ionos network plugin: Delete called", "resource_name", resource.GetName())

	namespace := k8sadapter.ComputeNamespace(&kresource.Scope{Tenant: resource.GetTenant()})
	name := resource.GetName()

	lan := &ionosv1alpha1.Lan{}
	err := n.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, lan)
	if err != nil {
		if apierrors.IsNotFound(err) {
			n.logger.Info("lan already gone", "namespace", namespace, "lan", name)
			return nil
		}
		n.logger.Error("failed to get lan before delete", "lan", name, "error", err)
		return err
	}

	if lan.GetDeletionTimestamp().IsZero() {
		n.logger.Info("deleting lan", "namespace", namespace, "lan", name)
		if err := n.client.Delete(ctx, lan); err != nil && !apierrors.IsNotFound(err) {
			n.logger.Error("failed to delete lan", "lan", name, "error", err)
			return err
		}
	}

	n.logger.Info("waiting for lan deletion", "namespace", namespace, "lan", name)
	return backend.ErrStillProcessing
}
