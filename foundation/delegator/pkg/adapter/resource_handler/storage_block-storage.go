package resourcehandler

import (
	"context"
	"errors"
	"log"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

/*
 *
 * NOTE: Code in this file is not complete nor ready for production usage.
 *       It's only for exploratory development.
 *
 */

type delegateFunc func(ctx context.Context, resource *regional.StorageBlockStorageDomain) error

type setStateFunc func(ctx context.Context, resource *regional.StorageBlockStorageDomain) error

type StorageBlockStorageCSPPlugin interface {
	Create(ctx context.Context, resource *regional.StorageBlockStorageDomain) error
	Delete(ctx context.Context, resource *regional.StorageBlockStorageDomain) error
	IncreaseSize(ctx context.Context, resource *regional.StorageBlockStorageDomain) error
	SetImage(ctx context.Context, resource *regional.StorageBlockStorageDomain) error
}

type StorageBlockStorageResourceHandler struct {
	port.GenericDelegatorResourceHandler[*regional.StorageBlockStorageDomain]
	repo   gateway_port.Repo[*regional.StorageBlockStorageDomain]
	plugin StorageBlockStorageCSPPlugin
}

var _ port.DelegatorResourceHandler[*regional.StorageBlockStorageDomain] = (*StorageBlockStorageResourceHandler)(nil)

func NewStorageBlockStorageResourceHandler(
	repo gateway_port.Repo[*regional.StorageBlockStorageDomain],
	plugin StorageBlockStorageCSPPlugin,
) *StorageBlockStorageResourceHandler {
	handler := &StorageBlockStorageResourceHandler{
		repo:   repo,
		plugin: plugin,
	}

	// Add admission rejection conditions
	handler.AddRejectionConditions(blockChangeSKU)
	handler.AddRejectionConditions(blockDecreaseSize)

	return handler
}

func (h *StorageBlockStorageResourceHandler) HandleReconcile(ctx context.Context, resource *regional.StorageBlockStorageDomain) error {
	// Find delegate operation which should be done.
	var delegate delegateFunc

	switch {
	case wantCreate(resource):
		delegate = h.plugin.Create

	case wantDelete(resource):
		delegate = h.plugin.Delete

	case wantIncreaseSize(resource):
		delegate = h.plugin.IncreaseSize

	case wantSetImage(resource):
		delegate = h.plugin.SetImage

	default:
		return nil // Nothing to do.
	}

	// Delegate the action to the CSP Plugin.
	if err := delegate(ctx, resource); err != nil {
		// Handle errors from the CSP Plugin.
		resource.Status.SetError(err)

		if err := h.repo.Update(ctx, resource); err != nil {
			return err // TODO: better error handling.
		}

		return nil
	}

	// Handle success of the delegated actions.
	var setState setStateFunc

	switch {
	case wantCreate(resource):
		resource.Status.StorageBlockStorageSpec = resource.Spec
		resource.Status.SetAvtive()

		setState = h.repo.Update

	case wantDelete(resource):
		setState = h.repo.Delete

	case wantIncreaseSize(resource):
		resource.Status.SizeGB = resource.Spec.SizeGB

		setState = h.repo.Update

	case wantSetImage(resource):
		resource.Status.SourceImageID = resource.Spec.SourceImageID

		setState = h.repo.Update

	default:
		log.Fatal("must never achieve that condition")
	}

	// Set the status of the resource properly.
	if err := setState(ctx, resource); err != nil {
		return err // TODO: better error handling.
	}

	return nil
}

//
// Admission Rejection Conditions

func blockChangeSKU(_ context.Context, resource *regional.StorageBlockStorageDomain) error {
	if resource.Status.State != model.StateCreating && resource.Status.SkuType != resource.Spec.SkuType {
		return errors.New("changing storage sku type is not allowed")
	}

	return nil
}

func blockDecreaseSize(_ context.Context, resource *regional.StorageBlockStorageDomain) error {
	if resource.Status.State != model.StateCreating && resource.Spec.SizeGB < resource.Status.SizeGB {
		return errors.New("decrease storage size is not allowed")
	}

	return nil
}

//
// Reconciliation Actions Conditions

func wantCreate(resource *regional.StorageBlockStorageDomain) bool {
	return resource.Status.State == model.StateCreating
}

func wantDelete(resource *regional.StorageBlockStorageDomain) bool {
	return resource.Status.State == model.StateDeleting
}

func wantIncreaseSize(resource *regional.StorageBlockStorageDomain) bool {
	return resource.Status.State == model.StateUpdating && resource.Spec.SizeGB > resource.Status.SizeGB
}

func wantSetImage(resource *regional.StorageBlockStorageDomain) bool {
	return resource.Status.State == model.StateUpdating && resource.Spec.SourceImageID != resource.Status.SourceImageID
}
