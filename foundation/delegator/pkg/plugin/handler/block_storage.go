package handler

import (
	"context"
	"errors"
	"log"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

type BlockStoragePluginHandler struct {
	GenericPluginHandler[*regional.BlockStorageDomain]
	repo   gateway.Repo[*regional.BlockStorageDomain]
	plugin plugin.BlockStorage
}

var _ delegator.PluginHandler[*regional.BlockStorageDomain] = (*BlockStoragePluginHandler)(nil)

func NewBlockStoragePluginHandler(
	repo gateway.Repo[*regional.BlockStorageDomain],
	plugin plugin.BlockStorage,
) *BlockStoragePluginHandler {
	handler := &BlockStoragePluginHandler{
		repo:   repo,
		plugin: plugin,
	}

	handler.AddRejectionConditions(blockDecreaseSize)

	return handler
}

//nolint:gocyclo // keep locality of behavior: the two switches describe the full reconciliation state machine in one place
func (h *BlockStoragePluginHandler) HandleReconcile(ctx context.Context, resource *regional.BlockStorageDomain) (bool, error) {
	var delegate delegator.DelegatedFunc[*regional.BlockStorageDomain]

	switch {
	case isBlockStorageAccepted(resource):
		delegate = BypassDelegated[*regional.BlockStorageDomain]

	case isBlockStoragePending(resource):
		delegate = BypassDelegated[*regional.BlockStorageDomain]

	case isBlockStorageCreating(resource):
		delegate = h.plugin.Create

	case wantBlockStorageDelete(resource):
		delegate = BypassDelegated[*regional.BlockStorageDomain]

	case isBlockStorageDeleting(resource):
		delegate = h.plugin.Delete

	case wantBlockStorageIncreaseSize(resource):
		delegate = BypassDelegated[*regional.BlockStorageDomain]

	case isBlockStorageIncreasingSize(resource):
		delegate = h.plugin.IncreaseSize

	case wantBlockStorageRetryCreate(resource) || wantBlockStorageRetryIncreaseSize(resource):
		delegate = BypassDelegated[*regional.BlockStorageDomain]

	default:
		return false, nil // Nothing to do.
	}

	if err := delegate(ctx, resource); err != nil {
		if errors.Is(err, delegator.ErrStillProcessing) {
			return true, nil
		}

		if requeue, err := h.setResourceErrorState(ctx, resource, err, false); err != nil {
			return requeue, err // TODO: better errors handling
		}

		return true, nil
	}

	switch {

	case isBlockStorageAccepted(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStatePending, false)

	case isBlockStoragePending(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating, true)

	case isBlockStorageCreating(resource):
		resource.Status.SizeGB = resource.Spec.SizeGB

		return h.setResourceState(ctx, resource, regional.ResourceStateActive, false)

	case wantBlockStorageDelete(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateDeleting, true)

	case isBlockStorageDeleting(resource):
		// Nothing to do: the delegator controller will remove the finalizers
		// in order to end the deletion process.
		return false, nil

	case wantBlockStorageIncreaseSize(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateUpdating, true)

	case isBlockStorageIncreasingSize(resource):
		resource.Status.SizeGB = resource.Spec.SizeGB

		return h.setResourceState(ctx, resource, regional.ResourceStateActive, false)

	case wantBlockStorageRetryCreate(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating, true)

	case wantBlockStorageRetryIncreaseSize(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateUpdating, true)

	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *BlockStoragePluginHandler) setResourceState(ctx context.Context, resource *regional.BlockStorageDomain, state regional.ResourceStateDomain, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &regional.BlockStorageStatusDomain{}
	}

	resource.Status.PushCondition(conditionFromState(state))
	for h.MaxConditions > 0 && len(resource.Status.Conditions) > h.MaxConditions {
		resource.Status.PopCondition()
	}

	if _, err := h.repo.UpdateStatus(ctx, resource); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return false, nil
		}

		return requeue, err
	}

	return requeue, nil
}

func (h *BlockStoragePluginHandler) setResourceErrorState(ctx context.Context, resource *regional.BlockStorageDomain, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &regional.BlockStorageStatusDomain{}
	}

	resource.Status.PushCondition(conditionFromError(err))
	for h.MaxConditions > 0 && len(resource.Status.Conditions) > h.MaxConditions {
		resource.Status.PopCondition()
	}

	if _, err := h.repo.UpdateStatus(ctx, resource); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return false, nil
		}

		return requeue, err
	}

	return requeue, nil
}

func blockDecreaseSize(_ context.Context, resource *regional.BlockStorageDomain) error {
	if resource.Status != nil &&
		resource.Status.State != regional.ResourceStateCreating &&
		resource.Spec.SizeGB < resource.Status.SizeGB {
		return errors.New("decrease storage size is not allowed")
	}

	return nil
}

func isBlockStorageAccepted(resource *regional.BlockStorageDomain) bool {
	return resource.Status == nil
}

func isBlockStoragePending(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == regional.ResourceStatePending)
}

func isBlockStorageCreating(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateCreating
}

func blockStorageIsNotDeleting(resource *regional.BlockStorageDomain) bool {
	return resource.Status == nil ||
		resource.Status.State != regional.ResourceStateDeleting
}

func wantBlockStorageDelete(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt != nil && blockStorageIsNotDeleting(resource)
}

func isBlockStorageDeleting(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateDeleting
}

func detectIncreaseSizeCondition(resource *regional.BlockStorageDomain) bool {
	return resource.Spec.SizeGB > resource.Status.SizeGB
}

func wantBlockStorageIncreaseSize(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateActive &&
		detectIncreaseSizeCondition(resource)
}

func isBlockStorageIncreasingSize(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateUpdating &&
		detectIncreaseSizeCondition(resource)
}

func wantBlockStorageRetryCreate(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateCreating
}

func wantBlockStorageRetryIncreaseSize(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateUpdating &&
		resource.Spec.SizeGB > resource.Status.SizeGB
}
