package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	repo "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/delegated"
	mutator_bypass "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/mutator"
	resolver_bypass "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/resolver"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter"
	repository "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository"
)

// Ensure BlockStorageHandler implements the BlockStorage interface
var _ plugin.BlockStorage = (*BlockStorageHandler)(nil)

// BlockStorageHandler handles BlockStorageDomain resources by interacting with Aruba BlockStorage.
// It is responsible for translating BlockStorageDomain resources to Aruba BlockStorage
// and managing their lifecycle (Create/Delete).
type BlockStorageHandler struct {
	wsRepository          repo.ReaderRepo[*regional.WorkspaceDomain]
	skuRepository         repo.ReaderRepo[*regional.StorageSKUDomain]
	bsRepository          repository.Repository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList]
	prjRepository         repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList]
	bsConverter           converter.Converter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage]
	wsConverter           converter.Converter[*regional.WorkspaceDomain, *v1alpha1.Project]
	createDelegated       *delegated.GenericDelegated[*regional.BlockStorageDomain, *SecaBlockStorageBundle, *ArubaBlockStorageBundle]
	deleteDelegated       *delegated.GenericDelegated[*regional.BlockStorageDomain, *SecaBlockStorageBundle, *ArubaBlockStorageBundle]
	increaseSizeDelegated *delegated.GenericDelegated[*regional.BlockStorageDomain, *SecaBlockStorageBundle, *ArubaBlockStorageBundle]
}

type SecaBlockStorageBundle struct {
	BlockStorage *regional.BlockStorageDomain
	Workspace    *regional.WorkspaceDomain
	StorageSku   *regional.StorageSKUDomain
}

type ArubaBlockStorageBundle struct {
	BlockStorage *v1alpha1.BlockStorage
	Project      *v1alpha1.Project
}

// NewBlockStorageHandler creates a new BlockStorageHandler with the provided repository and converter.
// It sets up the necessary delegated operations for creating and deleting WorkspaceDomain resources.
// The handler uses bypass mutators since no mutation is needed on the Aruba Project objects.
func NewBlockStorageHandler(
	wsRepo repo.ReaderRepo[*regional.WorkspaceDomain],
	skuRepo repo.ReaderRepo[*regional.StorageSKUDomain],
	bsRepo repository.Repository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList],
	prjRepo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList],
	bsConv converter.Converter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage],
	wsConv converter.Converter[*regional.WorkspaceDomain, *v1alpha1.Project]) *BlockStorageHandler {

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
			return p.BlockStorage.Status.Phase == v1alpha1.ResourcePhaseCreated
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

// Create creates a new BlockStorageDomain by creating an Aruba BlockStorage.
func (h *BlockStorageHandler) Create(ctx context.Context, resource *regional.BlockStorageDomain) error {
	return h.createDelegated.Do(ctx, resource)
}

// Delete deletes an existing BlockStorageDomain by deleting the corresponding Aruba BlockStorage.
func (h *BlockStorageHandler) Delete(ctx context.Context, resource *regional.BlockStorageDomain) error {
	return h.deleteDelegated.Do(ctx, resource)
}

// IncreaseSize increases the size of an existing BlockStorageDomain by updating the corresponding Aruba BlockStorage.
func (h *BlockStorageHandler) IncreaseSize(ctx context.Context, resource *regional.BlockStorageDomain) error {
	return h.increaseSizeDelegated.Do(ctx, resource)
}

func (h *BlockStorageHandler) checkBsDeleteCondition(resource *ArubaBlockStorageBundle) bool {
	// TODO: refactor design completely
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := h.bsRepository.Load(ctx, resource.BlockStorage)

	return apierrors.IsNotFound(err)
}

func (h *BlockStorageHandler) checkBsIncreaseSizeCondition(resource *ArubaBlockStorageBundle) bool {
	// TODO: refactor design completely
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	size := resource.BlockStorage.Spec.SizeGb

	err := h.bsRepository.Load(ctx, resource.BlockStorage)

	if err != nil {
		return false
	}

	return resource.BlockStorage.Spec.SizeGb == size && resource.BlockStorage.Status.Phase == v1alpha1.ResourcePhaseCreated
}

func (h *BlockStorageHandler) blockStorageMutateSizeFunc(mutable *ArubaBlockStorageBundle, params *SecaBlockStorageBundle) error {
	mutable.BlockStorage.Spec.SizeGb = int32(params.BlockStorage.Spec.SizeGB)

	return nil
}

func (h *BlockStorageHandler) BypassDependencyResolver(ctx context.Context, main *regional.BlockStorageDomain) (*SecaBlockStorageBundle, error) {
	return &SecaBlockStorageBundle{
		BlockStorage: main,
	}, nil
}

func (h *BlockStorageHandler) resolveSecaBlockStorageDependencies(ctx context.Context, resource *regional.BlockStorageDomain) (*SecaBlockStorageBundle, error) {
	ws := &regional.WorkspaceDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: resource.GetWorkspace(),
			},
			Scope: scope.Scope{
				Tenant: resource.GetTenant(),
			},
		},
	}

	err := h.wsRepository.Load(ctx, &ws)
	if err != nil {
		return nil, delegator.ErrStillProcessing // TODO: better error handling
	}

	if ws.Status == nil || ws.Status.State == nil || *ws.Status.State != regional.ResourceStateActive {
		return nil, delegator.ErrStillProcessing // TODO: better error handling
	}

	// TODO: this is a temporary solution, we should refactor the design to avoid this kind of parsing
	// issue https://github.com/eu-sovereign-cloud/ecp/issues/216
	splittedSKU := strings.Split(resource.Spec.SkuRef.Resource, "/")
	if len(splittedSKU) != 2 {
		return nil, errors.New("invalid SKU reference")
	}

	skuName := splittedSKU[1]

	storageSku := &regional.StorageSKUDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: skuName,
			},
			Scope: scope.Scope{
				Tenant: resource.GetTenant(),
			},
		},
	}

	err = h.skuRepository.Load(ctx, &storageSku)
	if err != nil {
		return nil, err // TODO: better error handling
	}

	return &SecaBlockStorageBundle{
		BlockStorage: resource,
		Workspace:    ws,
		StorageSku:   storageSku,
	}, nil

}

func (h *BlockStorageHandler) resolveArubaBlockStorageDependencies(ctx context.Context, resource *ArubaBlockStorageBundle) (*ArubaBlockStorageBundle, error) {
	err := h.prjRepository.Load(ctx, resource.Project)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, delegator.ErrStillProcessing // Project not found, wait for it to be created
		}

		return nil, err // Other errors should be returned for handling
	}

	if resource.Project.Status.Phase != v1alpha1.ResourcePhaseCreated {
		return nil, delegator.ErrStillProcessing // Project is not ready, wait for it to be active
	}

	return &ArubaBlockStorageBundle{
		BlockStorage: resource.BlockStorage,
		Project:      resource.Project,
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

// waitUntilManagedError waits until the provided condition is met for the given resource.
// If the condition is not met within the timeout, it returns delegator.ErrStillProcessing to indicate that the operation is still in progress.
func (h *BlockStorageHandler) waitUntilManagedError(ctx context.Context, resource *ArubaBlockStorageBundle, condition repository.WaitConditionFunc[*ArubaBlockStorageBundle]) (*ArubaBlockStorageBundle, error) {
	bs, err := h.bsRepository.WaitUntil(ctx, resource.BlockStorage, func(p *v1alpha1.BlockStorage) bool {
		return condition(&ArubaBlockStorageBundle{
			BlockStorage: p,
		})
	})

	if err != nil {
		// Check if the error is due to the resource not being found, which can be expected during deletion
		if apierrors.IsTimeout(err) {
			return nil, delegator.ErrStillProcessing // Resource is gone, treat as successful deletion
		}

		return nil, err // Return other errors for handling
	}

	return &ArubaBlockStorageBundle{
		BlockStorage: bs,
		Project:      resource.Project,
	}, nil
}
