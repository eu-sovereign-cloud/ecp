package handler

import (
	"context"
	"errors"
	"log"

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

func (h *BlockStoragePluginHandler) HandleReconcile(ctx context.Context, resource *regional.BlockStorageDomain) (bool, error) {
	var delegate delegator.DelegatedFunc[*regional.BlockStorageDomain]

	switch {
	case isBlockStoragePending(resource):
		delegate = BypassDelegated[*regional.BlockStorageDomain]

	case wantBlockStorageCreate(resource):
		delegate = h.plugin.Create

	case resource.DeletedAt != nil || wantBlockStorageDelete(resource):
		delegate = h.plugin.Delete

	case isBlockStorageActiveAndNeedsUpdate(resource):
		delegate = BypassDelegated[*regional.BlockStorageDomain]

	case isBlockStorageUpdatingToIncreaseSize(resource):
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
		if err := h.setResourceErrorState(ctx, resource, err); err != nil {
			return false, err // TODO: better errors handling
		}

		return true, nil
	}

	switch {
	case isBlockStoragePending(resource):
		return true, h.setResourceState(ctx, resource, regional.ResourceStateCreating)

	case wantBlockStorageCreate(resource):
		resource.Status.SizeGB = resource.Spec.SizeGB

		return false, h.setResourceState(ctx, resource, regional.ResourceStateActive)

	case wantBlockStorageDelete(resource):
		return false, h.repo.Delete(ctx, resource)

	case isBlockStorageActiveAndNeedsUpdate(resource):
		return true, h.setResourceState(ctx, resource, regional.ResourceStateUpdating)

	case isBlockStorageUpdatingToIncreaseSize(resource):
		resource.Status.SizeGB = resource.Spec.SizeGB

		return false, h.setResourceState(ctx, resource, regional.ResourceStateActive)

	case wantBlockStorageRetryCreate(resource):
		return true, h.setResourceState(ctx, resource, regional.ResourceStateCreating)

	case wantBlockStorageRetryIncreaseSize(resource):
		return true, h.setResourceState(ctx, resource, regional.ResourceStateUpdating)

	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *BlockStoragePluginHandler) setResourceState(ctx context.Context, resource *regional.BlockStorageDomain, state regional.ResourceStateDomain) error {
	// TODO: Why the BlockStorage Status is a pointer and the Workspace Status is a nasted structure?
	// ISSUE: https://github.com/eu-sovereign-cloud/ecp/issues/188
	if resource.Status == nil {
		resource.Status = &regional.BlockStorageStatus{}
	}

	resource.Status.State = &state

	if resource.Status.Conditions == nil {
		resource.Status.Conditions = []regional.StatusConditionDomain{}
	}

	resource.Status.Conditions = append(resource.Status.Conditions, conditionFromState(state))

	if _, err := h.repo.Update(ctx, resource); err != nil {
		return err // TODO: better error handling.
	}

	return nil
}

func (h *BlockStoragePluginHandler) setResourceErrorState(ctx context.Context, resource *regional.BlockStorageDomain, err error) error {
	// TODO: Why the BlockStorage Status is a pointer and the Workspace Status is a nasted structure?
	// ISSUE: https://github.com/eu-sovereign-cloud/ecp/issues/188
	if resource.Status == nil {
		resource.Status = &regional.BlockStorageStatus{}
	}

	state := regional.ResourceStateError
	resource.Status.State = &state

	if resource.Status.Conditions == nil {
		resource.Status.Conditions = []regional.StatusConditionDomain{}
	}

	resource.Status.Conditions = append(resource.Status.Conditions, conditionFromError(err))

	if _, err := h.repo.Update(ctx, resource); err != nil {
		return err // TODO: better error handling.
	}

	return nil
}

func blockDecreaseSize(_ context.Context, resource *regional.BlockStorageDomain) error {
	if resource.Status != nil &&
		resource.Status.State != nil &&
		*(resource.Status.State) != regional.ResourceStateCreating &&
		resource.Spec.SizeGB < resource.Status.SizeGB {
		return errors.New("decrease storage size is not allowed")
	}

	return nil
}

func isBlockStoragePending(resource *regional.BlockStorageDomain) bool {
	return resource.Status == nil || resource.Status.State == nil || *(resource.Status.State) == regional.ResourceStatePending
}

func wantBlockStorageCreate(resource *regional.BlockStorageDomain) bool {
	return resource.Status != nil && resource.Status.State != nil && *(resource.Status.State) == regional.ResourceStateCreating
}

func wantBlockStorageDelete(resource *regional.BlockStorageDomain) bool {
	return resource.Status != nil && resource.Status.State != nil && *(resource.Status.State) == regional.ResourceStateDeleting
}

func isBlockStorageActiveAndNeedsUpdate(resource *regional.BlockStorageDomain) bool {
	return resource.Status != nil && resource.Status.State != nil && *(resource.Status.State) == regional.ResourceStateActive && wantBlockStorageIncreaseSize(resource)
}

func isBlockStorageUpdatingToIncreaseSize(resource *regional.BlockStorageDomain) bool {
	return resource.Status != nil && resource.Status.State != nil && *(resource.Status.State) == regional.ResourceStateUpdating && wantBlockStorageIncreaseSize(resource)
}

func wantBlockStorageIncreaseSize(resource *regional.BlockStorageDomain) bool {
	return resource.Spec.SizeGB > resource.Status.SizeGB
}

func wantBlockStorageRetryCreate(resource *regional.BlockStorageDomain) bool {
	return resource.Status != nil &&
		resource.Status.State != nil &&
		*(resource.Status.State) == regional.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateCreating
}

func wantBlockStorageRetryIncreaseSize(resource *regional.BlockStorageDomain) bool {
	return resource.Status != nil &&
		resource.Status.State != nil &&
		*(resource.Status.State) == regional.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateUpdating &&
		resource.Spec.SizeGB > resource.Status.SizeGB
}
