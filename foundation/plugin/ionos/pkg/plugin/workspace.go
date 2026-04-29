package plugin

import (
	"context"
	"log/slog"

	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	v2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"

	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

const ProviderConfigName = "cluster-ionos-provider-config"
const ProviderConfigType = "ClusterProviderConfig"

type Workspace struct {
	client client.Client
	logger *slog.Logger
}

func NewWorkspace(client client.Client, logger *slog.Logger) *Workspace {
	return &Workspace{client: client, logger: logger}
}

func (w *Workspace) Create(ctx context.Context, resource *regional.WorkspaceDomain) error {
	w.logger.Info("ionos workspace plugin: Create called", "resource_name", resource.GetName())

	// Workspaces are tenant-scoped in ECP; compute namespace from tenant only.
	ownedNamespace := kubernetes.ComputeNamespace(&scope.Scope{Tenant: resource.GetTenant()})

	// Check whether the Ionos Datacenter already exists.
	existing := &ionosv1alpha1.Datacenter{}
	err := w.client.Get(ctx, client.ObjectKey{Namespace: ownedNamespace, Name: resource.GetName()}, existing)
	if err == nil {
		ready, err := checkReady(&existing.Status, "datacenter")
		if err != nil {
			w.logger.Error("datacenter in error state", "namespace", ownedNamespace, "datacenter", resource.GetName(), "error", err)
			return err
		}
		if ready {
			w.logger.Info("ionos datacenter is ready", "namespace", ownedNamespace, "datacenter", resource.GetName())
			return nil
		}
		w.logger.Info("ionos datacenter not yet ready, waiting", "namespace", ownedNamespace, "datacenter", resource.GetName())
		return delegator.ErrStillProcessing
	}
	if !apierrors.IsNotFound(err) {
		w.logger.Error("failed to check datacenter existence", "namespace", ownedNamespace, "datacenter", resource.GetName(), "error", err)
		return err
	}

	datacenter := &ionosv1alpha1.Datacenter{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ionosv1alpha1.CRDGroupVersion.String(),
			Kind:       ionosv1alpha1.Datacenter_Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.GetName(),
			Namespace: ownedNamespace,
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

	err = w.client.Create(ctx, datacenter)
	if err != nil {
		w.logger.Error("failed to create datacenter", "error", err)
		return err
	}

	w.logger.Info("datacenter created, waiting for it to be ready", "datacenter_name", datacenter.Name)
	return delegator.ErrStillProcessing
}

func (w *Workspace) Delete(ctx context.Context, resource *regional.WorkspaceDomain) error {
	w.logger.Info("ionos workspace plugin: Delete called", "resource_name", resource.GetName())

	// Keep namespace computation consistent with Create (tenant-scoped)
	ownedNamespace := kubernetes.ComputeNamespace(&scope.Scope{Tenant: resource.GetTenant()})
	key := client.ObjectKey{Name: resource.GetName(), Namespace: ownedNamespace}
	datacenter := &ionosv1alpha1.Datacenter{}

	err := w.client.Get(ctx, key, datacenter)
	if err != nil && !apierrors.IsNotFound(err) {
		w.logger.Error("failed to get datacenter before delete", "error", err)
		return err
	}

	if apierrors.IsNotFound(err) {
		w.logger.Info("datacenter already removed", "namespace", ownedNamespace, "datacenter_name", resource.GetName())
		return nil
	}

	if datacenter.GetDeletionTimestamp().IsZero() {
		datacenter.SetConditions(v1.Deleting())
		w.logger.Info("deleting datacenter", "namespace", ownedNamespace, "datacenter_name", resource.GetName())
		err = w.client.Delete(ctx, datacenter)
		if err != nil && !apierrors.IsNotFound(err) {
			w.logger.Error("failed to delete datacenter", "error", err)
			return err
		}
	}

	w.logger.Info("waiting for datacenter deletion", "namespace", ownedNamespace, "datacenter_name", resource.GetName())
	return delegator.ErrStillProcessing
}
