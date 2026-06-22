package crossplane

import (
	"context"
	"fmt"
	"log/slog"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	v2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes"
	netdom "github.com/eu-sovereign-cloud/ecp/resources/regional/network/networks/v1"
)

var _ port.NetworkStore = (*NetworkStore)(nil)

// NetworkStore implements the NetworkStore interface using Crossplane CRDs.
type NetworkStore struct {
	base
}

func NewNetworkStore(c client.Client, logger *slog.Logger) *NetworkStore {
	return &NetworkStore{base{client: c, logger: logger}}
}

func (a *NetworkStore) Create(ctx context.Context, domain *netdom.Network) error {
	desired := newLan(domain)

	existing := &ionosv1alpha1.Lan{}
	if err := a.client.Get(ctx, client.ObjectKeyFromObject(desired), existing); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		namespace := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})
		datacenter := &ionosv1alpha1.Datacenter{
			TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Datacenter_Kind},
			ObjectMeta: metav1.ObjectMeta{Name: domain.GetWorkspace(), Namespace: namespace},
		}
		if err := a.checkExisting(ctx, datacenter); err != nil {
			return fmt.Errorf("network %q requires workspace datacenter %q: %w", domain.GetName(), domain.GetWorkspace(), err)
		}
		return a.createCR(ctx, desired)
	}

	// Lan exists: if IPv6 enablement changed, push the update.
	// We compare nil vs non-nil only — not the specific CIDR — because IONOS
	// late-initializes spec.forProvider.ipv6CidrBlock from "AUTO" to the actual
	// assigned CIDR, so string equality would trigger an infinite update loop.
	if (desired.Spec.ForProvider.IPv6CidrBlock == nil) != (existing.Spec.ForProvider.IPv6CidrBlock == nil) {
		existing.Spec.ForProvider.IPv6CidrBlock = desired.Spec.ForProvider.IPv6CidrBlock
		return a.updateCR(ctx, existing)
	}

	return a.checkExisting(ctx, existing)
}

func (a *NetworkStore) Delete(ctx context.Context, domain *netdom.Network) error {
	namespace := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})
	return a.deleteCR(ctx, &ionosv1alpha1.Lan{
		TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Lan_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetName(), Namespace: namespace},
	})
}

func newLan(domain *netdom.Network) *ionosv1alpha1.Lan {
	namespace := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})
	lan := &ionosv1alpha1.Lan{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ionosv1alpha1.CRDGroupVersion.String(),
			Kind:       ionosv1alpha1.Lan_Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      domain.GetName(),
			Namespace: namespace,
		},
		Spec: ionosv1alpha1.LanSpec{
			ManagedResourceSpec: v2.ManagedResourceSpec{
				ProviderConfigReference: &v1.ProviderConfigReference{
					Name: ProviderConfigName,
					Kind: ProviderConfigType,
				},
			},
			ForProvider: ionosv1alpha1.LanParameters{
				Name: new(domain.GetName()),
				DatacenterIDRef: &v1.NamespacedReference{
					Name:      domain.GetWorkspace(),
					Namespace: namespace,
				},
				Public: new(false),
			},
		},
	}

	// A non-empty Cidr.IPv6 means "enable IPv6 on this LAN". We deliberately send
	// "AUTO" rather than passing domain.Spec.Cidr.IPv6 through verbatim:
	//
	// IONOS requires an explicitly-supplied LAN /64 to be a unique block that lies
	// *inside* the parent Datacenter's IPv6 CIDR. That Datacenter /56 is itself
	// auto-assigned by IONOS (workspace_store.go creates the Datacenter without an
	// IPv6CidrBlock), so at LAN-create time we have no way to know an in-range,
	// non-colliding /64. A tenant-supplied prefix would almost always be rejected.
	// "AUTO" lets IONOS allocate a valid /64 from the Datacenter block; it is then
	// late-initialized back onto spec.forProvider.ipv6CidrBlock (see Create, which
	// compares nil vs non-nil only to avoid fighting that late-init in a loop).
	//
	// To honor a specific requested prefix we would first have to control the
	// Datacenter's IPv6 /56 and sub-allocate /64s from it
	if domain.Spec.Cidr.IPv6 != "" {
		lan.Spec.ForProvider.IPv6CidrBlock = new("AUTO")
	}

	return lan
}
