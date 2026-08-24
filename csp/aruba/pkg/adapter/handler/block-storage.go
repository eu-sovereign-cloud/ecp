package handler

import (
	"context"
	"errors"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	ssdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"

	adaptconverter "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/delegated"
	mutator_bypass "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/mutator"
	resolver_bypass "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/resolver"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/skumap"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/converter"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/repository"
)

// Ensure BlockStorageHandler implements the BlockStorage interface
var _ bsk8s.BlockStoragePlugin = (*BlockStorageHandler)(nil)

// BlockStorageHandler handles BlockStorage resources by interacting with Aruba BlockStorage.
// It is responsible for translating BlockStorage resources to Aruba BlockStorage
// and managing their lifecycle (Create/Delete).
type BlockStorageHandler struct {
	wsRepository          persistence.ReaderRepo[*wsdom.Workspace]
	skuRepository         persistence.ReaderRepo[*ssdom.StorageSKU]
	bsRepository          repository.Repository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList]
	prjRepository         repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList]
	bsConverter           converter.Converter[*bsdom.BlockStorage, *v1alpha1.BlockStorage]
	wsConverter           converter.Converter[*wsdom.Workspace, *v1alpha1.Project]
	createDelegated       *delegated.GenericDelegated[*bsdom.BlockStorage, *SecaBlockStorageBundle, *ArubaBlockStorageBundle]
	deleteDelegated       *delegated.GenericDelegated[*bsdom.BlockStorage, *SecaBlockStorageBundle, *ArubaBlockStorageBundle]
	increaseSizeDelegated *delegated.GenericDelegated[*bsdom.BlockStorage, *SecaBlockStorageBundle, *ArubaBlockStorageBundle]
}

type SecaBlockStorageBundle struct {
	BlockStorage *bsdom.BlockStorage
	Workspace    *wsdom.Workspace
	StorageSku   *ssdom.StorageSKU
}

type ArubaBlockStorageBundle struct {
	BlockStorage *v1alpha1.BlockStorage
	Project      *v1alpha1.Project
}

// NewBlockStorageHandler creates a new BlockStorageHandler with the provided repository and converter.
// It sets up the necessary delegated operations for creating and deleting Workspace resources.
// The handler uses bypass mutators since no mutation is needed on the Aruba Project objects.
func NewBlockStorageHandler(
	wsRepo persistence.ReaderRepo[*wsdom.Workspace],
	skuRepo persistence.ReaderRepo[*ssdom.StorageSKU],
	bsRepo repository.Repository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList],
	prjRepo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList],
	bsConv converter.Converter[*bsdom.BlockStorage, *v1alpha1.BlockStorage],
	wsConv converter.Converter[*wsdom.Workspace, *v1alpha1.Project],
) *BlockStorageHandler {
	handler := &BlockStorageHandler{
		wsRepository:  wsRepo,
		skuRepository: skuRepo,
		bsRepository:  bsRepo,
		prjRepository: prjRepo,
		bsConverter:   bsConv,
		wsConverter:   wsConv,
	}

	handler.createDelegated = delegated.NewDelegated(
		handler.resolveSecaBlockStorageDependencies,
		handler.FromSECABundleToAruba,
		handler.resolveArubaBlockStorageDependencies,
		mutator_bypass.BypassMutateFunc[*ArubaBlockStorageBundle, *SecaBlockStorageBundle],
		handler.propagateCreate,
		handler.checkBsCreated,
	)

	handler.deleteDelegated = delegated.NewDelegated(
		handler.BypassDependencyResolver,
		handler.FromSECABundleToAruba,
		resolver_bypass.BypassResolveDependenciesFunc[*ArubaBlockStorageBundle],
		mutator_bypass.BypassMutateFunc[*ArubaBlockStorageBundle, *SecaBlockStorageBundle],
		handler.propagateDelete,
		handler.checkBsDeleted,
	)

	handler.increaseSizeDelegated = delegated.NewDelegated(
		handler.BypassDependencyResolver,
		handler.FromSECABundleToAruba,
		handler.resolveBlockStorageDependencies,
		handler.blockStorageMutateSizeFunc,
		handler.propagateUpdate,
		handler.checkBsResized,
	)

	return handler
}

// Create creates a new BlockStorage by creating an Aruba BlockStorage.
func (h *BlockStorageHandler) Create(ctx context.Context, domain *bsdom.BlockStorage) error {
	return h.createDelegated.Do(ctx, domain)
}

// Delete deletes an existing BlockStorage by deleting the corresponding Aruba BlockStorage.
func (h *BlockStorageHandler) Delete(ctx context.Context, domain *bsdom.BlockStorage) error {
	return h.deleteDelegated.Do(ctx, domain)
}

// IncreaseSize increases the size of an existing BlockStorage by updating the corresponding Aruba BlockStorage.
func (h *BlockStorageHandler) IncreaseSize(ctx context.Context, domain *bsdom.BlockStorage) error {
	return h.increaseSizeDelegated.Do(ctx, domain)
}

// checkBsCreated reports whether the Aruba BlockStorage already exists and has
// reached the active phase.
func (h *BlockStorageHandler) checkBsCreated(ctx context.Context, _ *SecaBlockStorageBundle, bundle *ArubaBlockStorageBundle) (bool, error) {
	observed := bundle.BlockStorage.DeepCopy()

	if err := h.bsRepository.Load(ctx, observed); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil // Not created yet, it must be created.
		}

		return false, err
	}

	return observed.Status.Phase == v1alpha1.ResourcePhaseActive, nil
}

// checkBsDeleted reports whether the Aruba BlockStorage is gone.
func (h *BlockStorageHandler) checkBsDeleted(ctx context.Context, _ *SecaBlockStorageBundle, bundle *ArubaBlockStorageBundle) (bool, error) {
	observed := bundle.BlockStorage.DeepCopy()

	if err := h.bsRepository.Load(ctx, observed); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil // Gone, deletion is complete.
		}

		return false, err
	}

	return false, nil // Still present, deletion is in progress.
}

// checkBsResized reports whether the Aruba BlockStorage already has the
// requested size and is back to the active phase.
func (h *BlockStorageHandler) checkBsResized(ctx context.Context, seca *SecaBlockStorageBundle, bundle *ArubaBlockStorageBundle) (bool, error) {
	desiredSize, err := adaptconverter.SecaToArubaSize(seca.BlockStorage.Spec.SizeGB)
	if err != nil {
		return false, err
	}

	observed := bundle.BlockStorage.DeepCopy()

	if err := h.bsRepository.Load(ctx, observed); err != nil {
		return false, err
	}

	return observed.Spec.SizeGB == desiredSize && observed.Status.Phase == v1alpha1.ResourcePhaseActive, nil
}

func (h *BlockStorageHandler) blockStorageMutateSizeFunc(mutable *ArubaBlockStorageBundle, params *SecaBlockStorageBundle) error {
	sizeGb, err := adaptconverter.SecaToArubaSize(params.BlockStorage.Spec.SizeGB)
	if err != nil {
		return err
	}
	mutable.BlockStorage.Spec.SizeGB = sizeGb

	return nil
}

func (h *BlockStorageHandler) BypassDependencyResolver(ctx context.Context, domain *bsdom.BlockStorage) (*SecaBlockStorageBundle, error) {
	return &SecaBlockStorageBundle{
		BlockStorage: domain,
	}, nil
}

func (h *BlockStorageHandler) resolveSecaBlockStorageDependencies(ctx context.Context, domain *bsdom.BlockStorage) (*SecaBlockStorageBundle, error) {
	ws := &wsdom.Workspace{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{
				Name: domain.GetWorkspace(),
			},
			Scope: res.Scope{
				Tenant: domain.GetTenant(),
			},
		},
	}

	err := h.wsRepository.Load(ctx, &ws)
	if err != nil {
		return nil, backend.StillProcessing // TODO: better error handling
	}

	if ws.Status == nil || ws.Status.State != commondomain.ResourceStateActive {
		return nil, backend.StillProcessing // TODO: better error handling
	}

	// A SKU is tenant-scoped, so the reference may name a tenant other than this block
	// storage's own — carried either as its own field or spelled out in the resource path
	// ("seca.storage/v1/tenants/<t>/skus/<name>"). ParseReference reads both and falls back
	// to this resource's tenant. See https://github.com/eu-sovereign-cloud/ecp/issues/216
	skuRef := commonbackend.ParseReference(domain.Spec.SkuRef, domain.GetTenant())
	if skuRef.Name == "" {
		return nil, errors.New("invalid SKU reference")
	}

	storageSku := &ssdom.StorageSKU{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{
				Name: skuRef.Name,
			},
			Scope: res.Scope{
				Tenant: skuRef.Tenant,
			},
		},
	}

	err = h.skuRepository.Load(ctx, &storageSku)
	if err != nil {
		return nil, err // TODO: better error handling
	}

	return &SecaBlockStorageBundle{
		BlockStorage: domain,
		Workspace:    ws,
		StorageSku:   storageSku,
	}, nil
}

func (h *BlockStorageHandler) resolveArubaBlockStorageDependencies(ctx context.Context, arubaBundle *ArubaBlockStorageBundle) (*ArubaBlockStorageBundle, error) {
	err := h.prjRepository.Load(ctx, arubaBundle.Project)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, backend.StillProcessing // Project not found, wait for it to be created
		}

		return nil, err // Other errors should be returned for handling
	}

	if arubaBundle.Project.Status.Phase != v1alpha1.ResourcePhaseActive {
		return nil, backend.StillProcessing // Project is not ready, wait for it to be active
	}

	return &ArubaBlockStorageBundle{
		BlockStorage: arubaBundle.BlockStorage,
		Project:      arubaBundle.Project,
	}, nil
}

func (h *BlockStorageHandler) FromSECABundleToAruba(from *SecaBlockStorageBundle) (*ArubaBlockStorageBundle, error) {
	response := &ArubaBlockStorageBundle{}

	if from.Workspace != nil {
		prj, err := h.wsConverter.FromSECAToAruba(from.Workspace)
		if err != nil {
			return nil, err // TODO: better error handling
		}

		response.Project = prj
	}

	bs, err := h.bsConverter.FromSECAToAruba(from.BlockStorage)
	if err != nil {
		return nil, err // TODO: better error handling
	}

	// Everything below is create-only, gated on the storage SKU because that is what tells the two
	// apart: the delete and resize chains resolve through BypassDependencyResolver and leave it
	// nil. Neither needs any of it - a delete addresses the volume by name and namespace, and a
	// resize reloads the live object over this one before updating it. Mapping the image on those
	// paths would be worse than useless: an image the table below does not know would fail the
	// convert step, which runs before propagate, leaving the volume impossible to delete.
	if from.StorageSku != nil {
		// Map the SECA storage SKU's capacity to an Aruba block-storage tier.
		bs.Spec.Type = skumap.StorageType(from.StorageSku.Spec.IOPS)

		// A block storage created from a source image is a boot disk: map the SECA image name to
		// the Aruba OS template code and mark the volume bootable, so Aruba installs an OS on it.
		// Without this a CloudServer booting from the volume is rejected by the CMP (semantic 400:
		// no bootable OS). Aruba has no image object, so the image is a no-op there - only its name
		// (the template code) matters, and it lands here on the volume.
		if ref := from.BlockStorage.Spec.SourceImageRef; ref != nil {
			image, err := skumap.ImageTemplate(lastSegment(ref.Resource))
			if err != nil {
				return nil, err
			}
			bs.Spec.Image = image
			bs.Spec.Bootable = true
		}
	}

	response.BlockStorage = bs

	return response, nil
}

// propagateCreate creates the Aruba BlockStorage. It is idempotent: because the
// create is (re)issued on every pass until the resource becomes active, an
// already existing resource is not treated as an error.
func (h *BlockStorageHandler) propagateCreate(ctx context.Context, from *ArubaBlockStorageBundle) error {
	if err := h.bsRepository.Create(ctx, from.BlockStorage); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// propagateDelete deletes the Aruba BlockStorage. It is idempotent: because the
// delete is (re)issued on every pass until the resource is gone, an already
// missing resource is not treated as an error.
func (h *BlockStorageHandler) propagateDelete(ctx context.Context, from *ArubaBlockStorageBundle) error {
	if err := h.bsRepository.Delete(ctx, from.BlockStorage); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

func (h *BlockStorageHandler) propagateUpdate(ctx context.Context, from *ArubaBlockStorageBundle) error {
	return h.bsRepository.Update(ctx, from.BlockStorage)
}

func (h *BlockStorageHandler) resolveBlockStorageDependencies(ctx context.Context, main *ArubaBlockStorageBundle) (*ArubaBlockStorageBundle, error) {
	err := h.bsRepository.Load(ctx, main.BlockStorage)

	return &ArubaBlockStorageBundle{
		BlockStorage: main.BlockStorage,
	}, err
}

// Update re-applies the BlockStorage's tags. Growing a volume is not handled here: it has its own
// plugin operation (IncreaseSize) and its own state transition, which the reconciler routes ahead
// of this one.
func (h *BlockStorageHandler) Update(ctx context.Context, domain *bsdom.BlockStorage) error {
	bs, err := h.bsConverter.FromSECAToAruba(domain)
	if err != nil {
		return err
	}

	return syncTags(ctx, h.bsRepository, bs, bs.Spec.Tags, func(b *v1alpha1.BlockStorage) *[]string { return &b.Spec.Tags })
}
