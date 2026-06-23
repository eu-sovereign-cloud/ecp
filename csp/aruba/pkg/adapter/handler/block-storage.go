package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	backend "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/storage/block-storages/v1"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resources/storage/block-storages/v1/backend/kubernetes"
	ssdom "github.com/eu-sovereign-cloud/ecp/resources/storage/storage-skus/v1"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/workspace/v1"

	adaptconverter "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/delegated"
	mutator_bypass "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/mutator"
	resolver_bypass "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/resolver"
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
		return nil, backend.ErrStillProcessing // TODO: better error handling
	}

	if ws.Status == nil || ws.Status.State != commondomain.ResourceStateActive {
		return nil, backend.ErrStillProcessing // TODO: better error handling
	}

	// The SKU reference may be a bare name ("sku-1") or a fully-qualified path
	// such as "seca.storage/v1/tenants/<t>/skus/<name>"; the SKU name is the
	// last path segment. See https://github.com/eu-sovereign-cloud/ecp/issues/216
	skuName := domain.Spec.SkuRef.Resource
	if idx := strings.LastIndex(skuName, "/"); idx != -1 {
		skuName = skuName[idx+1:]
	}
	if skuName == "" {
		return nil, errors.New("invalid SKU reference")
	}

	storageSku := &ssdom.StorageSKU{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{
				Name: skuName,
			},
			Scope: res.Scope{
				Tenant: domain.GetTenant(),
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
			return nil, backend.ErrStillProcessing // Project not found, wait for it to be created
		}

		return nil, err // Other errors should be returned for handling
	}

	if arubaBundle.Project.Status.Phase != v1alpha1.ResourcePhaseActive {
		return nil, backend.ErrStillProcessing // Project is not ready, wait for it to be active
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
