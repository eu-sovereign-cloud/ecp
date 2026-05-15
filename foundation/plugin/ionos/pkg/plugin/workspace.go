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

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"

	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

const ProviderConfigName = "cluster-ionos-provider-config"
const ProviderConfigType = "ClusterProviderConfig"

type Workspace struct {
	base
}

func NewWorkspace(c client.Client, logger *slog.Logger) *Workspace {
	return &Workspace{base{client: c, logger: logger}}
}

func newDatacenter(resource *regional.WorkspaceDomain) *ionosv1alpha1.Datacenter {
	namespace := kubernetes.ComputeNamespace(&scope.Scope{Tenant: resource.GetTenant()})
	return &ionosv1alpha1.Datacenter{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ionosv1alpha1.CRDGroupVersion.String(),
			Kind:       ionosv1alpha1.Datacenter_Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.GetName(),
			Namespace: namespace,
		},
		Spec: ionosv1alpha1.DatacenterSpec{
			ManagedResourceSpec: v2.ManagedResourceSpec{
				ProviderConfigReference: &v1.ProviderConfigReference{
					// todo move back to namespaced provider config once we can create users/tenants
					// which should create a namespaced provider config per workspace
					Name: ProviderConfigName,
					Kind: ProviderConfigType,
				},
			},
			ForProvider: ionosv1alpha1.DatacenterParameters{
				Name:        new(resource.GetName()),
				Description: new("Workspace: " + resource.GetName()),
				Location:    new("es/vit"), // Default location, should be configurable from region
			},
		},
	}
}

func (w *Workspace) Create(ctx context.Context, resource *regional.WorkspaceDomain) error {
	w.logger.Info("ionos workspace plugin: Create called", "resource_name", resource.GetName())

	datacenter := newDatacenter(resource)

	if err := w.client.Create(ctx, datacenter); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			w.logger.Error("failed to create datacenter", "error", err)
			return err
		}
		return w.checkExisting(ctx, datacenter)
	}

	w.logger.Info("datacenter created, waiting for it to be ready", "datacenter_name", datacenter.Name)
	return delegator.ErrStillProcessing
}

func (w *Workspace) Delete(ctx context.Context, resource *regional.WorkspaceDomain) error {
	w.logger.Info("ionos workspace plugin: Delete called", "resource_name", resource.GetName())

	// Keep namespace computation consistent with Create (tenant-scoped)
	ownedNamespace := kubernetes.ComputeNamespace(&scope.Scope{Tenant: resource.GetTenant()})
	datacenter := &ionosv1alpha1.Datacenter{
		ObjectMeta: metav1.ObjectMeta{Name: resource.GetName(), Namespace: ownedNamespace},
	}

	if err := w.client.Delete(ctx, datacenter); err != nil {
		if apierrors.IsNotFound(err) {
			w.logger.Info("datacenter already removed", "datacenter_name", resource.GetName())
			return nil
		}
		w.logger.Error("failed to delete datacenter", "error", err)
		return err
	}

	w.logger.Info("waiting for datacenter deletion", "datacenter_name", datacenter.Name)
	return delegator.ErrStillProcessing
}
