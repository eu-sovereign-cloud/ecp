package plugin

import (
	"context"
	"log/slog"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	v2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	"k8s.io/utils/ptr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

const ProviderConfigName = "ionos-provider-config"

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
	datacenter := &ionosv1alpha1.Datacenter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.GetName(),
			Namespace: resource.GetTenant(),
		},
		Spec: ionosv1alpha1.DatacenterSpec{
			ManagedResourceSpec: v2.ManagedResourceSpec{
				ProviderConfigReference: &v1.ProviderConfigReference{
					Name: ProviderConfigName,
				},
			},
			ForProvider: ionosv1alpha1.DatacenterParameters{
				Name:        ptr.To(resource.GetName()),
				Description: ptr.To("Workspace: " + resource.GetName()),
				Location:    ptr.To("de/txl"), // Default location, could be configurable
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

	datacenter := &ionosv1alpha1.Datacenter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.GetName(),
			Namespace: resource.GetTenant(),
		},
	}

	err := w.client.Delete(ctx, datacenter)
	if err != nil {
		w.logger.Error("failed to delete datacenter", "error", err)
		return err
	}

	w.logger.Info("datacenter deleted successfully", "datacenter_name", datacenter.Name)
	return nil
}

// Note: Workspace may not have an "IncreaseSize" equivalent; adjust based on actual interface
