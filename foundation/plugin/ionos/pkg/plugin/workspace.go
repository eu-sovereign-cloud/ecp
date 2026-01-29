package plugin

import (
	"context"
	"log/slog"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	v2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	"k8s.io/utils/ptr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

const ProviderConfigName = "cluster-ionos-provider-config"

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
	ns := kubernetes.ComputeNamespace(resource)
	if ns == "" {
		ns = "default"
	}

	// find the Workspace CR to use as owner
	ws := &workspacev1.Workspace{}
	if err := w.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: resource.GetName()}, ws); err != nil {
		if apierrors.IsNotFound(err) {
			w.logger.Error("workspace CR not found for ownerreference", "namespace", ns, "name", resource.GetName())
			return err
		}
		w.logger.Error("failed to get workspace CR", "error", err)
		return err
	}

	datacenter := &ionosv1alpha1.Datacenter{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ionosv1alpha1.CRDGroupVersion.String(),
			Kind:       ionosv1alpha1.Datacenter_Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.GetName(),
			Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: workspacev1.GroupVersion.String(),
					Kind:       workspacev1.Kind,
					Name:       ws.GetName(),
					UID:        ws.GetUID(),
					Controller: ptr.To(true),
					// Ensure the Workspace cannot be deleted until the Datacenter is gone.
					BlockOwnerDeletion: ptr.To(true),
				},
			},
		},
		Spec: ionosv1alpha1.DatacenterSpec{
			ManagedResourceSpec: v2.ManagedResourceSpec{
				ProviderConfigReference: &v1.ProviderConfigReference{
					// todo move back to namespaced provider config once we can create users/tenants
					// which should create a namespaced provider config per workspace
					Name: ProviderConfigName,
					Kind: "ClusterProviderConfig",
				},
			},
			ForProvider: ionosv1alpha1.DatacenterParameters{
				Name:        ptr.To(resource.GetName()),
				Description: ptr.To("Workspace: " + resource.GetName()),
				Location:    ptr.To("de/txl"), // Default location, should be configurable from region
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

	ns := kubernetes.ComputeNamespace(resource)
	if ns == "" {
		ns = "default"
	}

	key := client.ObjectKey{Name: resource.GetName(), Namespace: ns}

	// Issue a delete to ensure Crossplane starts finalization / external cleanup.
	datacenter := &ionosv1alpha1.Datacenter{}
	if err := w.client.Get(ctx, key, datacenter); err != nil {
		if apierrors.IsNotFound(err) {
			// Already gone.
			return nil
		}
		w.logger.Error("failed to get datacenter before delete", "error", err)
		return err
	}
	datacenter.SetConditions(v1.Deleting())
	if err := w.client.Delete(ctx, datacenter); err != nil {
		// If it's already gone, treat as success.
		if !apierrors.IsNotFound(err) {
			w.logger.Error("failed to delete datacenter", "error", err)
			return err
		}
	}

	// const (
	// 	pollEvery     = 2 * time.Second
	// 	deleteTimeout = 10 * time.Minute
	// )
	//
	// // Wait until the Datacenter resource is fully removed from the API.
	// // This implicitly waits for Crossplane finalizers (e.g. finalizer.managedresource.crossplane.io)
	// // to complete and thus also blocks Workspace/namespace deletion until external cleanup is done.
	// if err := wait.PollUntilContextTimeout(ctx, pollEvery, deleteTimeout, true, func(ctx context.Context) (bool, error) {
	// 	err := w.client.Get(ctx, key, &ionosv1alpha1.Datacenter{})
	// 	if apierrors.IsNotFound(err) {
	// 		return true, nil
	// 	}
	// 	if err != nil {
	// 		return false, err
	// 	}
	// 	w.logger.Info("waiting for datacenter deletion", "namespace", ns, "datacenter_name", resource.GetName())
	// 	return false, nil
	// }); err != nil {
	// 	// Make timeouts explicit; these typically mean Crossplane/provider couldn't remove finalizers.
	// 	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, wait.ErrWaitTimeout) {
	// 		w.logger.Error("timed out waiting for datacenter deletion (likely stuck on Crossplane finalizer)", "namespace", ns, "datacenter_name", resource.GetName(), "error", err)
	// 		return err
	// 	}
	// 	w.logger.Error("failed while waiting for datacenter deletion", "error", err)
	// 	return err
	// }

	w.logger.Info("datacenter deleted successfully", "namespace", ns, "datacenter_name", resource.GetName())
	return nil
}
