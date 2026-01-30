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

	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

type BlockStorage struct {
	client client.Client
	logger *slog.Logger
}

func NewBlockStorage(client client.Client, logger *slog.Logger) *BlockStorage {
	return &BlockStorage{client: client, logger: logger}
}

func (b *BlockStorage) Create(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("ionos block storage plugin: Create called", "resource_name", resource.GetName())

	// Map ECP BlockStorage to Crossplane Volume
	volume := &ionosv1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.GetName(),
			Namespace: "crossplane-system", // Adjust namespace as needed
		},
		Spec: ionosv1alpha1.VolumeSpec{
			ForProvider: ionosv1alpha1.VolumeParameters_2{
				DatacenterIDSelector: &v1.NamespacedSelector{
					MatchLabels: map[string]string{},
					Namespace:   "",
				}, // todo: link to datacenter created in workspace
				Name:             ptr.To(resource.Name),
				Size:             ptr.To(float64(resource.Spec.SizeGB)),
				DiskType:         ptr.To("HDD"),
				AvailabilityZone: ptr.To("AUTO"),
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

	err := b.client.Create(ctx, volume)
	if err != nil {
		b.logger.Error("failed to create volume", "error", err)
		return err
	}

	b.logger.Info("volume created successfully", "volume_name", volume.Name)
	return nil
}

func (b *BlockStorage) Delete(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("ionos block storage plugin: Delete called", "resource_name", resource.GetName())

	volume := &ionosv1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.GetName(),
			Namespace: "crossplane-system",
		},
	}

	if err := b.client.Get(ctx, client.ObjectKeyFromObject(volume), volume); err != nil {
		if apierrors.IsNotFound(err) {
			b.logger.Info("volume already gone", "volume_name", volume.Name)
			return nil
		}
		b.logger.Error("failed to get volume before delete", "error", err)
		return err
	}

	if volume.GetDeletionTimestamp().IsZero() {
		b.logger.Info("deleting volume", "volume_name", volume.Name)
		if err := b.client.Delete(ctx, volume); err != nil {
			if !apierrors.IsNotFound(err) {
				b.logger.Error("failed to delete volume", "error", err)
				return err
			}
		}
	}

	b.logger.Info("waiting for volume deletion", "volume_name", volume.Name)
	return delegator.ErrStillProcessing
}

func (b *BlockStorage) IncreaseSize(ctx context.Context, resource *regional.BlockStorageDomain) error {
	b.logger.Info("ionos block storage plugin: IncreaseSize called", "resource_name", resource.GetName(), "new_size_gb", resource.Spec.SizeGB)

	// Fetch existing volume
	volume := &ionosv1alpha1.Volume{}
	err := b.client.Get(ctx, client.ObjectKey{Name: resource.GetName(), Namespace: "crossplane-system"}, volume)
	if err != nil {
		b.logger.Error("failed to get volume", "error", err)
		return err
	}

	// Update size
	volume.Spec.ForProvider.Size = ptr.To(float64(resource.Spec.SizeGB))

	err = b.client.Update(ctx, volume)
	if err != nil {
		b.logger.Error("failed to update volume size", "error", err)
		return err
	}

	b.logger.Info("volume size increased successfully", "volume_name", volume.Name)
	return nil
}
