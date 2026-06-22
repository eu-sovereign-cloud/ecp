package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	backend "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/backend/kubernetes"
	ssdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/storage-skus/v1"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1"

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
	wsConv converter.Converter[*wsdom.Workspace, *v1alpha1.Project]) *BlockStorageHandler {

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
		func(p *ArubaBlockStorageBundle) bool {
			return p.BlockStorage.Status.Phase == v1alpha1.ResourcePhaseActive
		},
		handler.waitUntilManagedError,
	)

	handler.deleteDelegated = delegated.NewDelegated(
		handler.BypassDependencyResolver,
		handler.FromSECABundleToAruba,
		resolver_bypass.BypassResolveDependenciesFunc[*ArubaBlockStorageBundle],
		mutator_bypass.BypassMutateFunc[*ArubaBlockStorageBundle, *SecaBlockStorageBundle],
		handler.propagateDelete,
		handler.checkBsDeleteCondition,
		handler.waitUntilManagedError,
	)

	handler.increaseSizeDelegated = delegated.NewDelegated(
		handler.BypassDependencyResolver,
		handler.FromSECABundleToAruba,
		handler.resolveBlockStorageDependencies,
		handler.blockStorageMutateSizeFunc,
		handler.propagateUpdate,
		handler.checkBsIncreaseSizeCondition,
		handler.waitUntilManagedError,
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

func (h *BlockStorageHandler) checkBsDeleteCondition(arubaBundle *ArubaBlockStorageBundle) bool {
	// TODO: refactor design completely
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := h.bsRepository.Load(ctx, arubaBundle.BlockStorage)

	return apierrors.IsNotFound(err)
}

func (h *BlockStorageHandler) checkBsIncreaseSizeCondition(arubaBundle *ArubaBlockStorageBundle) bool {
	// TODO: refactor design completely
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	size := arubaBundle.BlockStorage.Spec.SizeGB

	err := h.bsRepository.Load(ctx, arubaBundle.BlockStorage)

	if err != nil {
		return false
	}

	return arubaBundle.BlockStorage.Spec.SizeGB == size && arubaBundle.BlockStorage.Status.Phase == v1alpha1.ResourcePhaseActive
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

	// TODO: this is a temporary solution, we should refactor the design to avoid this kind of parsing
	// issue https://github.com/eu-sovereign-cloud/ecp/issues/216
	splittedSKU := strings.Split(domain.Spec.SkuRef.Resource, "/")
	if len(splittedSKU) != 2 {
		return nil, errors.New("invalid SKU reference")
	}

	skuName := splittedSKU[1]

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
	var response = &ArubaBlockStorageBundle{}

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

func (h *BlockStorageHandler) propagateCreate(ctx context.Context, from *ArubaBlockStorageBundle) error {
	return h.bsRepository.Create(ctx, from.BlockStorage)
}

func (h *BlockStorageHandler) propagateDelete(ctx context.Context, from *ArubaBlockStorageBundle) error {
	return h.bsRepository.Delete(ctx, from.BlockStorage)
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

// waitUntilManagedError waits until the provided condition is met for the given arubaBundle.
// If the condition is not met within the timeout, it returns backend.ErrStillProcessing to indicate that the operation is still in progress.
func (h *BlockStorageHandler) waitUntilManagedError(ctx context.Context, arubaBundle *ArubaBlockStorageBundle, condition repository.WaitConditionFunc[*ArubaBlockStorageBundle]) (*ArubaBlockStorageBundle, error) {
	bs, err := h.bsRepository.WaitUntil(ctx, arubaBundle.BlockStorage, func(p *v1alpha1.BlockStorage) bool {
		return condition(&ArubaBlockStorageBundle{
			BlockStorage: p,
		})
	})

	if err != nil {
		// Check if the error is due to the resource not being found, which can be expected during deletion
		if apierrors.IsTimeout(err) {
			return nil, backend.ErrStillProcessing // Resource is gone, treat as successful deletion
		}

		return nil, err // Return other errors for handling
	}

	return &ArubaBlockStorageBundle{
		BlockStorage: bs,
		Project:      arubaBundle.Project,
	}, nil
}
