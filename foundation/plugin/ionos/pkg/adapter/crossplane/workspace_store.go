package crossplane

import (
	"context"
	"log/slog"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	v2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8s "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/pkg/port"
)

var _ port.WorkspaceStore = (*WorkspaceStore)(nil)

// WorkspaceStore implements the Workspace interface using Crossplane CRDs.
type WorkspaceStore struct {
	base
}

func NewWorkspaceStore(c client.Client, logger *slog.Logger) *WorkspaceStore {
	return &WorkspaceStore{base{client: c, logger: logger}}
}

func (a *WorkspaceStore) Create(ctx context.Context, domain *regional.WorkspaceDomain) error {
	return a.createCR(ctx, newDatacenter(domain))
}

func (a *WorkspaceStore) Delete(ctx context.Context, domain *regional.WorkspaceDomain) error {
	namespace := k8s.ComputeNamespace(&scope.Scope{Tenant: domain.GetTenant()})
	return a.deleteCR(ctx, &ionosv1alpha1.Datacenter{
		TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Datacenter_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetName(), Namespace: namespace},
	})
}

func newDatacenter(domain *regional.WorkspaceDomain) *ionosv1alpha1.Datacenter {
	namespace := k8s.ComputeNamespace(&scope.Scope{Tenant: domain.GetTenant()})
	return &ionosv1alpha1.Datacenter{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ionosv1alpha1.CRDGroupVersion.String(),
			Kind:       ionosv1alpha1.Datacenter_Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      domain.GetName(),
			Namespace: namespace,
		},
		Spec: ionosv1alpha1.DatacenterSpec{
			ManagedResourceSpec: v2.ManagedResourceSpec{
				ProviderConfigReference: &v1.ProviderConfigReference{
					Name: providerConfigName,
					Kind: providerConfigType,
				},
			},
			ForProvider: ionosv1alpha1.DatacenterParameters{
				Name:        new(domain.GetName()),
				Description: new("Workspace: " + domain.GetName()),
				Location:    new("es/vit"),
			},
		},
	}
}
