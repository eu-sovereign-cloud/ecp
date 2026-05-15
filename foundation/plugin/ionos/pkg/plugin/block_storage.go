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

	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
	k8s "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

type BlockStorage struct {
	base
}

func NewBlockStorage(c client.Client, logger *slog.Logger) *BlockStorage {
	return &BlockStorage{base{client: c, logger: logger}}
}

func newVolume(resource *regional.BlockStorageDomain) *ionosv1alpha1.Volume {
	namespace := k8s.ComputeNamespace(&scope.Scope{Tenant: resource.GetTenant()})
	return &ionosv1alpha1.Volume{
		TypeMeta: metav1.TypeMeta{Kind: ionosv1alpha1.Volume_Kind},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.GetName(),
			Namespace: namespace,
		},
		Spec: ionosv1alpha1.VolumeSpec{
			ForProvider: ionosv1alpha1.VolumeParameters_2{
				DatacenterIDRef: &v1.NamespacedReference{
					Name:      resource.GetWorkspace(),
					Namespace: namespace,
				},
				Name: new(resource.Name),
				Size: new(float64(resource.Spec.SizeGB)),
				// todo access sku ref to retrieve block storage type
				DiskType:         new("SSD"),
				AvailabilityZone: new("AUTO"),
				// todo access image ref to retrieve image
				ImageName: new("ubuntu:22.04"),
				// todo access attached server to retrieve ssh key
				ImagePassword: new("dummyPw123"),
			},
			ManagedResourceSpec: v2.ManagedResourceSpec{
				ProviderConfigReference: &v1.ProviderConfigReference{
					// todo move back to namespaced provider config once we can create users/tenants
					// which should create a namespaced provider config per workspace
					Name: ProviderConfigName,
					Kind: ProviderConfigType,
				},
			},
		},
	}
}

func (b *BlockStorage) Create(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("ionos block storage plugin: Create called", "resource_name", resource.GetName())

	b.logger.Info("block storage skuRef",
		"region", resource.Spec.SkuRef.Region,
		"tenant", resource.Spec.SkuRef.Tenant, "ws", resource.Spec.SkuRef.Workspace,
		"provider", resource.Spec.SkuRef.Provider, "resource", resource.Spec.SkuRef.Resource)

	volume := newVolume(resource)

	if err := b.client.Create(ctx, volume); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			b.logger.Error("failed to create volume", "error", err)
			return err
		}
		return b.checkExisting(ctx, volume)
	}

	b.logger.Info("volume created, waiting for it to be ready", "volume_name", volume.Name)
	return delegator.ErrStillProcessing
}

func (b *BlockStorage) Delete(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("ionos block storage plugin: Delete called", "resource_name", resource.GetName())
	namespace := k8s.ComputeNamespace(&scope.Scope{Tenant: resource.GetTenant()})

	volume := &ionosv1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.GetName(),
			Namespace: namespace,
		},
	}

	if err := b.client.Delete(ctx, volume); err != nil {
		if apierrors.IsNotFound(err) {
			b.logger.Info("volume already gone", "volume_name", volume.Name)
			return nil
		}
		b.logger.Error("failed to delete volume", "error", err)
		return err
	}

	b.logger.Info("waiting for volume deletion", "volume_name", volume.Name)
	return delegator.ErrStillProcessing
}

func (b *BlockStorage) IncreaseSize(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("ionos block storage plugin: IncreaseSize called", "resource_name", resource.GetName(), "new_size_gb", resource.Spec.SizeGB)
	namespace := k8s.ComputeNamespace(&scope.Scope{Tenant: resource.GetTenant()})

	volume := &ionosv1alpha1.Volume{}
	err := b.client.Get(ctx, client.ObjectKey{Name: resource.GetName(), Namespace: namespace}, volume)
	if err != nil {
		b.logger.Error("failed to get volume", "name", resource.GetName(), "namespace", namespace, "error", err)
		return err
	}

	volume.Spec.ForProvider.Size = new(float64(resource.Spec.SizeGB))

	err = b.client.Update(ctx, volume)
	if err != nil {
		b.logger.Error("failed to update volume size", "name", resource.GetName(), "namespace", namespace, "error", err)
		return err
	}

	b.logger.Info("volume size increased successfully", "volume_name", volume.Name, "namespace", namespace)
	return nil
}
