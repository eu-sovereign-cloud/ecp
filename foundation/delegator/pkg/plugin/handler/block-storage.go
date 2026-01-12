package handler

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	delegator_port "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

type BlockStorageResourceHandler struct {
	GenericDelegatorResourceHandler[*regional.BlockStorageDomain]
	repo   gateway_port.Repo[*regional.BlockStorageDomain]
	plugin plugin.BlockStorage
}

var _ delegator_port.ResourceHandler[*regional.BlockStorageDomain] = (*BlockStorageResourceHandler)(nil)

func NewBlockStorageResourceHandler(
	repo gateway_port.Repo[*regional.BlockStorageDomain],
	plugin plugin.BlockStorage,
) *BlockStorageResourceHandler {
	handler := &BlockStorageResourceHandler{
		repo:   repo,
		plugin: plugin,
	}

	// Add admission rejection conditions
	handler.AddRejectionConditions(blockDecreaseSize)

	return handler
}

func (h *BlockStorageResourceHandler) HandleReconcile(ctx context.Context, resource *regional.BlockStorageDomain) error {
	// Find delegate operation which should be done.
	var delegate delegator_port.DelegatedFunc[*regional.BlockStorageDomain]

	switch {
	case isPending(resource):
		delegate = BypassDelegated[*regional.BlockStorageDomain]

	case wantCreate(resource):
		delegate = h.plugin.Create

	case wantDelete(resource):
		delegate = h.plugin.Delete

	case wantIncreaseSize(resource):
		delegate = h.plugin.IncreaseSize

	case wantRetryCreate(resource) || wantRetryIncreaseSize(resource):
		delegate = BypassDelegated[*regional.BlockStorageDomain]

	default:
		return nil // Nothing to do.
	}

	// Delegate the action to the CSP Plugin.
	if err := delegate(ctx, resource); err != nil {
		// Handle errors from the CSP Plugin.
		state := regional.ResourceStateError
		resource.Status.State = &state

		resource.Status.Conditions = append(resource.Status.Conditions, regional.StatusCondition{
			LastTransitionAt: time.Now(),
			Message:          err.Error(),
			State:            state,
		})

		if _, err := h.repo.Update(ctx, resource); err != nil {
			return err // TODO: better error handling.
		}

		return nil
	}

	switch {
	case isPending(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating)

	case wantCreate(resource):
		resource.Status.SizeGB = resource.Spec.SizeGB

		return h.setResourceState(ctx, resource, regional.ResourceStateActive)

	case wantDelete(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateDeleting)

	case wantIncreaseSize(resource):
		resource.Status.SizeGB = resource.Spec.SizeGB

		return h.setResourceState(ctx, resource, regional.ResourceStateActive)

	case wantRetryCreate(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating)

	case wantRetryIncreaseSize(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateUpdating)

	default:
		log.Fatal("must never achieve that condition")
	}

	return nil
}

//
// Helper Methods

func (h *BlockStorageResourceHandler) setResourceState(ctx context.Context, resource *regional.BlockStorageDomain, state regional.ResourceState) error {
	resource.Status.State = &state

	resource.Status.Conditions = append(resource.Status.Conditions, regional.StatusCondition{
		LastTransitionAt: time.Now(),
		State:            state,
	})

	if _, err := h.repo.Update(ctx, resource); err != nil {
		return err // TODO: better error handling.
	}

	return nil
}

//
// Admission Rejection Conditions

func blockDecreaseSize(_ context.Context, resource *regional.BlockStorageDomain) error {
	if *(resource.Status.State) != regional.ResourceStateCreating && resource.Spec.SizeGB < resource.Status.SizeGB {
		return errors.New("decrease storage size is not allowed")
	}

	return nil
}

//
// Reconciliation Actions Conditions

func isPending(resource *regional.BlockStorageDomain) bool {
	return *(resource.Status.State) == regional.ResourceStatePending
}

func wantCreate(resource *regional.BlockStorageDomain) bool {
	return *(resource.Status.State) == regional.ResourceStateCreating
}

func wantDelete(resource *regional.BlockStorageDomain) bool {
	return *(resource.Status.State) == regional.ResourceStateDeleting
}

func wantIncreaseSize(resource *regional.BlockStorageDomain) bool {
	return *(resource.Status.State) == regional.ResourceStateUpdating && resource.Spec.SizeGB > resource.Status.SizeGB
}

func wantRetryCreate(resource *regional.BlockStorageDomain) bool {
	return *(resource.Status.State) == regional.ResourceStateError &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateCreating
}

func wantRetryIncreaseSize(resource *regional.BlockStorageDomain) bool {
	return *(resource.Status.State) == regional.ResourceStateError &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateUpdating &&
		resource.Spec.SizeGB > resource.Status.SizeGB
}
