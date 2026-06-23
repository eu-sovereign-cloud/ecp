package kubernetes

import (
	"context"
	"errors"
	"log"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

	frameworkbackend "github.com/eu-sovereign-cloud/ecp/framework/backend"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resources/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/storage/block-storages/v1"
)

// BlockStoragePluginHandler drives the block-storage reconciliation state machine.
type BlockStoragePluginHandler struct {
	frameworkbackend.GenericPluginHandler[*bsdom.BlockStorage]
	repo   persistence.Repo[*bsdom.BlockStorage]
	plugin BlockStoragePlugin
}

var _ backendport.PluginHandler[*bsdom.BlockStorage] = (*BlockStoragePluginHandler)(nil)

// NewBlockStoragePluginHandler creates a new BlockStoragePluginHandler.
func NewBlockStoragePluginHandler(
	repo persistence.Repo[*bsdom.BlockStorage],
	plugin BlockStoragePlugin,
	maxConditions int,
) *BlockStoragePluginHandler {
	handler := &BlockStoragePluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions
	handler.AddRejectionConditions(blockDecreaseSize)

	return handler
}

//nolint:gocyclo // keep locality of behavior: the two switches describe the full reconciliation state machine in one place
func (h *BlockStoragePluginHandler) HandleReconcile(ctx context.Context, resource *bsdom.BlockStorage) (bool, error) {
	var delegate backendport.DelegatedFunc[*bsdom.BlockStorage]

	switch {
	case isBlockStorageAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*bsdom.BlockStorage]

	case isBlockStoragePending(resource):
		delegate = frameworkbackend.BypassDelegated[*bsdom.BlockStorage]

	case isBlockStorageCreating(resource):
		delegate = h.plugin.Create

	case wantBlockStorageDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*bsdom.BlockStorage]

	case isBlockStorageDeleting(resource):
		delegate = h.plugin.Delete

	case wantBlockStorageIncreaseSize(resource):
		delegate = frameworkbackend.BypassDelegated[*bsdom.BlockStorage]

	case isBlockStorageIncreasingSize(resource):
		delegate = h.plugin.IncreaseSize

	case wantBlockStorageRetryCreate(resource) || wantBlockStorageRetryIncreaseSize(resource):
		delegate = frameworkbackend.BypassDelegated[*bsdom.BlockStorage]

	default:
		return false, nil // Nothing to do.
	}

	if err := delegate(ctx, resource); err != nil {
		if errors.Is(err, backendport.ErrStillProcessing) {
			return true, nil
		}

		if requeue, err := h.setResourceErrorState(ctx, resource, err, false); err != nil {
			return requeue, err
		}

		return true, nil
	}

	switch {

	case isBlockStorageAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)

	case isBlockStoragePending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)

	case isBlockStorageCreating(resource):
		resource.Status.SizeGB = resource.Spec.SizeGB

		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)

	case wantBlockStorageDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)

	case isBlockStorageDeleting(resource):
		// Nothing to do: the controller will remove the finalizers to end the deletion process.
		return false, nil

	case wantBlockStorageIncreaseSize(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateUpdating, true)

	case isBlockStorageIncreasingSize(resource):
		resource.Status.SizeGB = resource.Spec.SizeGB

		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)

	case wantBlockStorageRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)

	case wantBlockStorageRetryIncreaseSize(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateUpdating, true)

	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *BlockStoragePluginHandler) setResourceState(ctx context.Context, resource *bsdom.BlockStorage, state commondomain.ResourceStateDomain, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &bsdom.BlockStorageStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	for h.MaxConditions > 0 && len(resource.Status.Conditions) > h.MaxConditions {
		resource.Status.PopCondition()
	}

	if _, err := h.repo.UpdateStatus(ctx, resource); err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return false, nil
		}

		return requeue, err
	}

	return requeue, nil
}

func (h *BlockStoragePluginHandler) setResourceErrorState(ctx context.Context, resource *bsdom.BlockStorage, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &bsdom.BlockStorageStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	for h.MaxConditions > 0 && len(resource.Status.Conditions) > h.MaxConditions {
		resource.Status.PopCondition()
	}

	if _, updateErr := h.repo.UpdateStatus(ctx, resource); updateErr != nil {
		if errors.Is(updateErr, kernel.ErrNotFound) {
			return false, nil
		}

		return requeue, updateErr
	}

	return requeue, nil
}

func blockDecreaseSize(_ context.Context, resource *bsdom.BlockStorage) error {
	if resource.Status != nil &&
		resource.Status.State != commondomain.ResourceStateCreating &&
		resource.Spec.SizeGB < resource.Status.SizeGB {
		return errors.New("decrease storage size is not allowed")
	}

	return nil
}

func isBlockStorageAccepted(resource *bsdom.BlockStorage) bool {
	return resource.Status == nil
}

func isBlockStoragePending(resource *bsdom.BlockStorage) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
}

func isBlockStorageCreating(resource *bsdom.BlockStorage) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func blockStorageIsNotDeleting(resource *bsdom.BlockStorage) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantBlockStorageDelete(resource *bsdom.BlockStorage) bool {
	return resource.DeletedAt != nil && blockStorageIsNotDeleting(resource)
}

func isBlockStorageDeleting(resource *bsdom.BlockStorage) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func detectIncreaseSizeCondition(resource *bsdom.BlockStorage) bool {
	return resource.Spec.SizeGB > resource.Status.SizeGB
}

func wantBlockStorageIncreaseSize(resource *bsdom.BlockStorage) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive &&
		detectIncreaseSizeCondition(resource)
}

func isBlockStorageIncreasingSize(resource *bsdom.BlockStorage) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateUpdating &&
		detectIncreaseSizeCondition(resource)
}

func wantBlockStorageRetryCreate(resource *bsdom.BlockStorage) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == commondomain.ResourceStateCreating
}

func wantBlockStorageRetryIncreaseSize(resource *bsdom.BlockStorage) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == commondomain.ResourceStateUpdating &&
		resource.Spec.SizeGB > resource.Status.SizeGB
}
