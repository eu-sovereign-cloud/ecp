package handler

import (
	"context"
	"time"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"

	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/delegated"
	resolver_bypass "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/resolver"

	mutator_bypass "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/mutator"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter"
	repository "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository"
	"k8s.io/apimachinery/pkg/api/errors"
)

// Ensure BlockStorageHandler implements the BlockStorage interface
var _ plugin.BlockStorage = (*BlockStorageHandler)(nil)

// BlockStorageHandler handles BlockStorageDomain resources by interacting with Aruba BlockStorage.
// It is responsible for translating BlockStorageDomain resources to Aruba BlockStorage
// and managing their lifecycle (Create/Delete).
type BlockStorageHandler struct {
	repository            repository.Repository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList]
	converter             converter.Converter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage]
	createDelegated       *delegated.GenericDelegated[*regional.BlockStorageDomain, *regional.BlockStorageDomain, *v1alpha1.BlockStorage]
	deleteDelegated       *delegated.GenericDelegated[*regional.BlockStorageDomain, *regional.BlockStorageDomain, *v1alpha1.BlockStorage]
	increaseSizeDelegated *delegated.GenericDelegated[*regional.BlockStorageDomain, *regional.BlockStorageDomain, *v1alpha1.BlockStorage]
}

// NewBlockStorageHandler creates a new BlockStorageHandler with the provided repository and converter.
// It sets up the necessary delegated operations for creating and deleting WorkspaceDomain resources.
// The handler uses bypass mutators since no mutation is needed on the Aruba Project objects.
func NewBlockStorageHandler(repo repository.Repository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], conv converter.Converter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage]) *BlockStorageHandler {
	bsHandler := &BlockStorageHandler{
		repository: repo,
		converter:  conv,
	}

	bsHandler.createDelegated = delegated.NewStraightDelegated(
		conv.FromSECAToAruba,
		mutator_bypass.BypassMutateFunc[*v1alpha1.BlockStorage, *regional.BlockStorageDomain],
		repo.Create,
		func(p *v1alpha1.BlockStorage) bool {
			return p.Status.Phase == v1alpha1.ResourcePhaseCreated
		},
		repo.WaitUntil,
	)

	bsHandler.deleteDelegated = delegated.NewStraightDelegated(
		conv.FromSECAToAruba,
		mutator_bypass.BypassMutateFunc[*v1alpha1.BlockStorage, *regional.BlockStorageDomain],
		repo.Delete,
		bsHandler.checkBsDeleteCondition,
		repo.WaitUntil,
	)

	bsHandler.increaseSizeDelegated = delegated.NewDelegated(
		resolver_bypass.BypassResolveDependenciesFunc[*regional.BlockStorageDomain],
		conv.FromSECAToAruba,
		bsHandler.ResolveBlockStorageDependencies,
		BlockStorageMutateSizeFunc,
		repo.Update,
		bsHandler.checkBsIncreaseSizeCondition,
		repo.WaitUntil,
	)

	return bsHandler
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

func (h *BlockStorageHandler) checkBsDeleteCondition(resource *v1alpha1.BlockStorage) bool {
	//TODO: refactor design completely
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := h.repository.Load(ctx, resource)

	return errors.IsNotFound(err)
}

func (h *BlockStorageHandler) checkBsIncreaseSizeCondition(resource *v1alpha1.BlockStorage) bool {
	//TODO: refactor design completely
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	size := resource.Spec.SizeGb

	err := h.repository.Load(ctx, resource)

	if err != nil {
		return false
	}

	return resource.Spec.SizeGb == size && resource.Status.Phase == v1alpha1.ResourcePhaseCreated
}

func BlockStorageMutateSizeFunc(
	mutable *v1alpha1.BlockStorage,
	params *regional.BlockStorageDomain,
) error {
	mutable.Spec.SizeGb = int32(params.Spec.SizeGB)

	return nil
}

func (h *BlockStorageHandler) ResolveBlockStorageDependencies(ctx context.Context, main *v1alpha1.BlockStorage) (*v1alpha1.BlockStorage, error) {
	err := h.repository.Load(ctx, main)

	return main, err

}
