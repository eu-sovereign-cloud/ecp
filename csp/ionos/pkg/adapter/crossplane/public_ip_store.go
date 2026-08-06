package crossplane

import (
	"context"
	"fmt"
	"log/slog"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	v2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

var _ port.PublicIPStore = (*PublicIPStore)(nil)

type PublicIPStore struct {
	base
}

func NewPublicIPStore(c client.Client, logger *slog.Logger) *PublicIPStore {
	return &PublicIPStore{base{client: c, logger: logger}}
}

func (a *PublicIPStore) Create(ctx context.Context, domain *publicipdom.PublicIp) error {
	ipb, err := newIPBlock(domain)
	if err != nil {
		return err
	}
	return a.createCR(ctx, ipb)
}

func (a *PublicIPStore) Delete(ctx context.Context, domain *publicipdom.PublicIp) error {
	namespace := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})
	return a.deleteCR(ctx, &ionosv1alpha1.Ipblock{
		TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Ipblock_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetName(), Namespace: namespace},
	})
}

func newIPBlock(domain *publicipdom.PublicIp) (*ionosv1alpha1.Ipblock, error) {
	location, err := translateLocation(domain.Region)
	if err != nil {
		return nil, err
	}
	namespace := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})
	return &ionosv1alpha1.Ipblock{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ionosv1alpha1.CRDGroupVersion.String(),
			Kind:       ionosv1alpha1.Ipblock_Kind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetName(), Namespace: namespace},
		Spec: ionosv1alpha1.IpblockSpec{
			ForProvider: ionosv1alpha1.IpblockParameters{
				Name:     new(domain.GetName()),
				Location: new(location),
				Size:     new(float64(1)),
			},
			ManagedResourceSpec: v2.ManagedResourceSpec{
				ProviderConfigReference: &v1.ProviderConfigReference{
					Name: ProviderConfigName,
					Kind: ProviderConfigType,
				},
			},
		},
	}, nil
}

// readReservedIP returns the first public IP reserved on the IPBlock named `name`.
// Returns ErrStillProcessing until the provider has assigned an address, or the
// reconcile error if the provider has given up (e.g. region out of addresses),
// so a failed IPBlock surfaces as an instance error instead of requeuing forever.
func readReservedIP(ctx context.Context, c client.Client, namespace, name string) (string, error) {
	ipb := &ionosv1alpha1.Ipblock{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, ipb); err != nil {
		return "", fmt.Errorf("read ipblock %q: %w", name, err)
	}
	if err := reconcileError(ipb); err != nil {
		return "", fmt.Errorf("ipblock %q: %w", name, err)
	}
	ips := ipb.Status.AtProvider.Ips
	if len(ips) == 0 || ips[0] == nil || *ips[0] == "" {
		return "", backend.ErrStillProcessing
	}
	return *ips[0], nil
}
