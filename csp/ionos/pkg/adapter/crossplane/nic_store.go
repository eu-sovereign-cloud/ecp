package crossplane

import (
	"context"
	"log/slog"

	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

var _ port.NicStore = (*NicStore)(nil)

type NicStore struct {
	base
}

func NewNicStore(c client.Client, logger *slog.Logger) *NicStore {
	return &NicStore{base{client: c, logger: logger}}
}

// Create observes the instance-owned IONOS Nic. The real Nic (attached to the Server, on the
// public LAN, with the reserved public IP) is created by the Instance plugin at PowerOn using
// this NIC's name, so before power-on there is nothing to provision and the CR is a ready
// declaration (observer). Once the instance is powered on we observe the real Nic's state.
func (a *NicStore) Create(ctx context.Context, domain *nicdom.Nic) error {
	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})
	nic := &ionosv1alpha1.Nic{
		TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Nic_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetName(), Namespace: ns},
	}
	if err := a.client.Get(ctx, client.ObjectKeyFromObject(nic), nic); err != nil {
		if apierrors.IsNotFound(err) {
			a.logger.Info("nic: ready as declaration, provisioned at instance power-on",
				"namespace", ns, "nic", domain.GetName())
			return nil
		}
		return err
	}
	return a.checkExisting(ctx, nic)
}

// Delete is a no-op: the IONOS Nic is instance-owned (created and torn down together with the
// Server at power-on/off by the Instance plugin), so deleting the SECA NIC observer must not
// touch it.
func (a *NicStore) Delete(ctx context.Context, domain *nicdom.Nic) error {
	return nil
}
