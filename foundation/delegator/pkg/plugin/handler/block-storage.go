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
	case isBlockStorageAccepted(resource):
		log.Println("-->DETECT isBlockStorageAccepted", "resource", resource)
		delegate = BypassDelegated[*regional.BlockStorageDomain]
	case isBlockStoragePending(resource):
		log.Println("-->DETECT isBlockStoragePending", "resource", resource)
		delegate = BypassDelegated[*regional.BlockStorageDomain]

	case isBlockStorageCreating(resource):
		log.Println("-->DETECT isBlockStorageCreating", "resource", resource)
		delegate = h.plugin.Create

	case wantBlockStorageDelete(resource):
		log.Println("-->DETECT wantBlockStorageDelete", "resource", resource)
		// Transition to deleting state before calling plugin.Delete
		delegate = BypassDelegated[*regional.BlockStorageDomain]

	case isBlockStorageDeleting(resource):
		log.Println("-->DETECT isBlockStorageDeleting", "resource", resource)
		delegate = h.plugin.Delete

	case wantBlockStorageIncreaseSize(resource):
		log.Println("-->DETECT wantBlockStorageIncreaseSize", "resource", resource)
		delegate = BypassDelegated[*regional.BlockStorageDomain]

	case isBlockStorageIncreasingSize(resource):
		log.Println("-->DETECT isBlockStorageIncreasingSize", "resource", resource)
		delegate = h.plugin.IncreaseSize

	case wantBlockStorageRetryCreate(resource) || wantBlockStorageRetryIncreaseSize(resource):
		log.Println("-->DETECT wantBlockStorageRetryCreate or wantBlockStorageRetryIncreaseSize", "resource", resource)
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

	case isBlockStorageAccepted(resource):
		log.Println("-->REACT isBlockStorageAccepted", "resource", resource)
		return false, h.setResourceState(ctx, resource, regional.ResourceStatePending)
	case isBlockStoragePending(resource):
		log.Println("-->REACT isBlockStoragePending", "resource", resource)
		return true, h.setResourceState(ctx, resource, regional.ResourceStateCreating)

	case isBlockStorageCreating(resource):
		log.Println("-->REACT isBlockStorageCreating", "resource", resource)
		resource.Status.SizeGB = resource.Spec.SizeGB

		return false, h.setResourceState(ctx, resource, regional.ResourceStateActive)

	case wantBlockStorageDelete(resource):
		log.Println("-->REACT wantBlockStorageDelete (setting state to Deleting)", "resource", resource)
		return true, h.setResourceState(ctx, resource, regional.ResourceStateDeleting)

	case isBlockStorageDeleting(resource):
		log.Println("-->REACT isBlockStorageDeleting", "resource", resource)
		return false, nil // Let the controller handle finalizer removal and actual deletion from K8s

	case wantBlockStorageIncreaseSize(resource):
		log.Println("-->REACT wantBlockStorageIncreaseSize", "resource", resource)
		return true, h.setResourceState(ctx, resource, regional.ResourceStateUpdating)

	case isBlockStorageIncreasingSize(resource):
		log.Println("-->REACT isBlockStorageIncreasingSize", "resource", resource)
		resource.Status.SizeGB = resource.Spec.SizeGB

		return false, h.setResourceState(ctx, resource, regional.ResourceStateActive)

	case wantBlockStorageRetryCreate(resource):
		log.Println("-->REACT wantBlockStorageRetryCreate", "resource", resource)
		return true, h.setResourceState(ctx, resource, regional.ResourceStateCreating)

	case wantBlockStorageRetryIncreaseSize(resource):
		log.Println("-->REACT wantBlockStorageRetryIncreaseSize", "resource", resource)
		return true, h.setResourceState(ctx, resource, regional.ResourceStateUpdating)

	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *BlockStoragePluginHandler) setResourceState(ctx context.Context, resource *regional.BlockStorageDomain, state regional.ResourceStateDomain) error {
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

func isBlockStorageAccepted(resource *regional.BlockStorageDomain) bool {
	return resource.Status == nil || resource.Status.State == nil
}

func isBlockStoragePending(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt == nil && (resource.Status == nil || resource.Status.State == nil || *(resource.Status.State) == regional.ResourceStatePending)
}

func isBlockStorageCreating(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt == nil && resource.Status != nil && resource.Status.State != nil && *(resource.Status.State) == regional.ResourceStateCreating
}

// wantBlockStorageDelete detects when K8s deletion has been triggered (DeletedAt is set)
// but the resource state hasn't been transitioned to Deleting yet.
func wantBlockStorageDelete(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt != nil && (resource.Status == nil || resource.Status.State == nil || *(resource.Status.State) != regional.ResourceStateDeleting)
}

// isBlockStorageDeleting detects when K8s deletion has been triggered (DeletedAt is set).
func isBlockStorageDeleting(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt != nil && resource.Status != nil && resource.Status.State != nil && *(resource.Status.State) == regional.ResourceStateDeleting
}

func wantBlockStorageIncreaseSize(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt == nil && resource.Status != nil && resource.Status.State != nil && *(resource.Status.State) == regional.ResourceStateActive && detectIncreaseSizeCondition(resource)
}

func isBlockStorageIncreasingSize(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt == nil && resource.Status != nil && resource.Status.State != nil && *(resource.Status.State) == regional.ResourceStateUpdating && detectIncreaseSizeCondition(resource)
}

func detectIncreaseSizeCondition(resource *regional.BlockStorageDomain) bool {
	return resource.Spec.SizeGB > resource.Status.SizeGB
}

func wantBlockStorageRetryCreate(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State != nil &&
		*(resource.Status.State) == regional.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateCreating
}

func wantBlockStorageRetryIncreaseSize(resource *regional.BlockStorageDomain) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State != nil &&
		*(resource.Status.State) == regional.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateUpdating &&
		resource.Spec.SizeGB > resource.Status.SizeGB
}
