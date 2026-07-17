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
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

var _ port.BlockStorageStore = (*BlockStorageStore)(nil)

type BlockStorageStore struct {
	base
}

func NewBlockStorageStore(c client.Client, logger *slog.Logger) *BlockStorageStore {
	return &BlockStorageStore{base{client: c, logger: logger}}
}

func (a *BlockStorageStore) Create(ctx context.Context, domain *bsdom.BlockStorage) error {
	namespace := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})

	// Image-backed volumes need SSH keys + user-data (only the Instance has them),
	// so the Instance plugin creates them at PowerOn. Here we only observe.
	if domain.Spec.SourceImageRef != nil {
		vol := &ionosv1alpha1.Volume{
			TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Volume_Kind},
			ObjectMeta: metav1.ObjectMeta{Name: domain.GetName(), Namespace: namespace},
		}
		if err := a.client.Get(ctx, client.ObjectKeyFromObject(vol), vol); err != nil {
			if apierrors.IsNotFound(err) {
				a.logger.Info("image-backed volume not provisioned yet, waiting for instance",
					"namespace", namespace, "volume", domain.GetName())
				return backend.ErrStillProcessing
			}
			return err
		}
		return a.checkExisting(ctx, vol)
	}

	// Data volume: create it independently (no image, no SSH keys).
	datacenter := &ionosv1alpha1.Datacenter{
		TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Datacenter_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetWorkspace(), Namespace: namespace},
	}
	if err := a.checkExisting(ctx, datacenter); err != nil {
		return fmt.Errorf("block storage %q requires workspace datacenter %q: %w", domain.GetName(), domain.GetWorkspace(), err)
	}
	return a.createCR(ctx, newVolume(domain))
}

func (a *BlockStorageStore) Delete(ctx context.Context, domain *bsdom.BlockStorage) error {
	namespace := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})
	return a.deleteCR(ctx, &ionosv1alpha1.Volume{
		TypeMeta:   metav1.TypeMeta{Kind: ionosv1alpha1.Volume_Kind},
		ObjectMeta: metav1.ObjectMeta{Name: domain.GetName(), Namespace: namespace},
	})
}

func (a *BlockStorageStore) IncreaseSize(ctx context.Context, domain *bsdom.BlockStorage) error {
	namespace := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})
	vol := &ionosv1alpha1.Volume{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: domain.GetName(), Namespace: namespace}, vol); err != nil {
		a.logger.Error("failed to get volume", "name", domain.GetName(), "error", err)
		return err
	}
	desiredSize := float64(domain.Spec.SizeGB)
	if vol.Spec.ForProvider.Size != nil && *vol.Spec.ForProvider.Size == desiredSize {
		return a.checkExisting(ctx, vol)
	}
	vol.Spec.ForProvider.Size = new(desiredSize)
	return a.updateCR(ctx, vol)
}

func newVolume(domain *bsdom.BlockStorage) *ionosv1alpha1.Volume {
	namespace := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: domain.GetTenant()})
	return &ionosv1alpha1.Volume{
		TypeMeta: metav1.TypeMeta{Kind: ionosv1alpha1.Volume_Kind},
		ObjectMeta: metav1.ObjectMeta{
			Name:      domain.GetName(),
			Namespace: namespace,
		},
		Spec: ionosv1alpha1.VolumeSpec{
			ForProvider: ionosv1alpha1.VolumeParameters_2{
				DatacenterIDRef: &v1.NamespacedReference{
					Name:      domain.GetWorkspace(),
					Namespace: namespace,
				},
				Name:             new(domain.GetName()),
				Size:             new(float64(domain.Spec.SizeGB)),
				DiskType:         new("SSD"),
				AvailabilityZone: new("AUTO"),
			},
			ManagedResourceSpec: v2.ManagedResourceSpec{
				ProviderConfigReference: &v1.ProviderConfigReference{
					Name: ProviderConfigName,
					Kind: ProviderConfigType,
				},
			},
		},
	}
}
