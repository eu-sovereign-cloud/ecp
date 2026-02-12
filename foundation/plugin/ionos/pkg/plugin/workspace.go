package plugin

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

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

	// Map ECP Workspace to Crossplane Datacenter (logical grouping of resources)
	// ownerNamespace := kubernetes.ComputeNamespace(&scope.Scope{Tenant: resource.Tenant})
	ownedNamespace := kubernetes.ComputeNamespace(&scope.Scope{Tenant: resource.Tenant, Workspace: resource.GetName()})

	// can't reference the workspace CR as owner because it's in a different namespace, and cross-namespace owner references are not allowed in Kubernetes
	// ws := &workspacev1.Workspace{}
	// if err := w.client.Get(ctx, client.ObjectKey{Namespace: ownerNamespace, Name: resource.GetName()}, ws); err != nil {
	// 	if apierrors.IsNotFound(err) {
	// 		w.logger.Error("workspace CR not found", "namespace", ownerNamespace, "name", resource.GetName())
	// 		return err
	// 	}
	// 	w.logger.Error("failed to get workspace CR", "error", err)
	// 	return err
	// }

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ownedNamespace,
			Labels: map[string]string{labels.InternalTenantLabel: resource.GetTenant(), labels.InternalWorkspaceLabel: resource.GetName()},
		},
	}
	if err := w.client.Create(ctx, ns); err != nil {
		w.logger.Error("failed to create namespace for owner workspace", "workspace", resource.GetName(), "tenant", resource.GetTenant(), "error", err)
		return err
	}
	w.logger.Info("namespace created successfully", "namespace", ownedNamespace)

	datacenter := &ionosv1alpha1.Datacenter{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ionosv1alpha1.CRDGroupVersion.String(),
			Kind:       ionosv1alpha1.Datacenter_Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.GetName(),
			Namespace: ownedNamespace,
			// can't reference the workspace CR as owner because it's in a different namespace, and cross-namespace owner references are not allowed in Kubernetes
			// OwnerReferences: []metav1.OwnerReference{
			// 	{
			// 		APIVersion: workspacev1.GroupVersion.String(),
			// 		Kind:       workspacev1.Kind,
			// 		Name:       ws.GetName(),
			// 		UID:        ws.GetUID(),
			// 		Controller: ptr.To(true),
			// 		// Ensure the Workspace cannot be deleted until the Datacenter is gone.
			// 		BlockOwnerDeletion: ptr.To(true),
			// 	},
			// },
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
				Name:        ptr.To(resource.GetName()),
				Description: ptr.To("Workspace: " + resource.GetName()),
				Location:    ptr.To("es/vit"), // Default location, should be configurable from region
			},
		},
	}

	err := w.client.Create(ctx, datacenter)
	if err != nil {
		w.logger.Error("failed to create datacenter", "error", err)
		return err
	}

	w.logger.Info("datacenter created successfully", "datacenter_name", datacenter.Name)
	return nil
}

func (w *Workspace) Delete(ctx context.Context, resource *regional.WorkspaceDomain) error {
	w.logger.Info("ionos workspace plugin: Delete called", "resource_name", resource.GetName())

	ownedNamespace := kubernetes.ComputeNamespace(&scope.Scope{Tenant: resource.Tenant, Workspace: resource.GetName()})
	key := client.ObjectKey{Name: resource.GetName(), Namespace: ownedNamespace}
	datacenter := &ionosv1alpha1.Datacenter{}

	err := w.client.Get(ctx, key, datacenter)
	if err != nil && !apierrors.IsNotFound(err) {
		w.logger.Error("failed to get datacenter before delete", "error", err)
		return err
	}

	if apierrors.IsNotFound(err) {
		w.logger.Info("datacenter already removed", "namespace", ownedNamespace, "datacenter_name", resource.GetName())
		err = w.client.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ownedNamespace}})
		if err != nil && !apierrors.IsNotFound(err) {
			w.logger.Error("failed to delete namespace owned by workspace", "workspace", resource.GetName(), "tenant", resource.GetTenant(), "error", err)
			return err
		}
		w.logger.Info("namespace owned by workspace already removed", "workspace", resource.GetName(), "namespace", ownedNamespace)
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
