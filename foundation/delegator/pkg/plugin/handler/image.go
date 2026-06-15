package handler

import (
	"context"
	"errors"
	"log"

	model "github.com/eu-sovereign-cloud/ecp/foundation/models"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

// defaultImageSizeMB is the placeholder size reported in the Image status once the
// dummy reconcile flow finishes creating the resource. A real CSP would derive this
// from the referenced block storage.
const defaultImageSizeMB = 1024

type ImagePluginHandler struct {
	GenericPluginHandler[*regional.ImageDomain]
	repo   gateway.Repo[*regional.ImageDomain]
	plugin plugin.Image
}

var _ delegator.PluginHandler[*regional.ImageDomain] = (*ImagePluginHandler)(nil)

func NewImagePluginHandler(
	repo gateway.Repo[*regional.ImageDomain],
	plugin plugin.Image,
) *ImagePluginHandler {
	handler := &ImagePluginHandler{
		repo:   repo,
		plugin: plugin,
	}

	handler.AddRejectionConditions(rejectImageMutation)

	return handler
}

// rejectImageMutation enforces image immutability. Per the SECA image-handling
// specification an image is immutable once created: to change it, the existing
// image must be deleted and a new (ideally versioned) image created. This guard
// rejects any modification to an image that has already been created (Active).
//
// NOTE: like blockDecreaseSize, this is a single-object rejection condition; it
// keys off the observed status because the reconciled status is the only
// applied-state signal available at admission time. Relaxing this to allow
// mutable label/annotation edits while still blocking spec changes would require
// the admission layer to supply the previous object for an old-vs-new diff.
func rejectImageMutation(_ context.Context, resource *regional.ImageDomain) error {
	if resource.Status != nil && resource.Status.State == regional.ResourceStateActive {
		return errors.New("image is immutable once created: delete it and create a new image instead of updating it")
	}

	return nil
}

// HandleReconcile drives an Image through a simple create/delete lifecycle. Unlike
// block storage there is no resize, so the state machine only covers
// Pending -> Creating -> Active and the deletion path.
func (h *ImagePluginHandler) HandleReconcile(ctx context.Context, resource *regional.ImageDomain) (bool, error) {
	var delegate delegator.DelegatedFunc[*regional.ImageDomain]

	switch {
	case isImageAccepted(resource):
		delegate = BypassDelegated[*regional.ImageDomain]

	case isImagePending(resource):
		delegate = BypassDelegated[*regional.ImageDomain]

	case isImageCreating(resource):
		delegate = h.plugin.Create

	case wantImageDelete(resource):
		delegate = BypassDelegated[*regional.ImageDomain]

	case isImageDeleting(resource):
		delegate = h.plugin.Delete

	case wantImageRetryCreate(resource):
		delegate = BypassDelegated[*regional.ImageDomain]

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

	case isImageAccepted(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStatePending, false)

	case isImagePending(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating, true)

	case isImageCreating(resource):
		size := defaultImageSizeMB
		resource.Status.SizeMB = &size

		return h.setResourceState(ctx, resource, regional.ResourceStateActive, false)

	case wantImageDelete(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateDeleting, true)

	case isImageDeleting(resource):
		// Nothing to do: the delegator controller will remove the finalizers
		// in order to end the deletion process.
		return false, nil

	case wantImageRetryCreate(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating, true)

	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *ImagePluginHandler) setResourceState(ctx context.Context, resource *regional.ImageDomain, state regional.ResourceStateDomain, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &regional.ImageStatusDomain{}
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

func (h *ImagePluginHandler) setResourceErrorState(ctx context.Context, resource *regional.ImageDomain, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &regional.ImageStatusDomain{}
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

func isImageAccepted(resource *regional.ImageDomain) bool {
	return resource.Status == nil
}

func isImagePending(resource *regional.ImageDomain) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == regional.ResourceStatePending)
}

func isImageCreating(resource *regional.ImageDomain) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateCreating
}

func imageIsNotDeleting(resource *regional.ImageDomain) bool {
	return resource.Status == nil ||
		resource.Status.State != regional.ResourceStateDeleting
}

func wantImageDelete(resource *regional.ImageDomain) bool {
	return resource.DeletedAt != nil && imageIsNotDeleting(resource)
}

func isImageDeleting(resource *regional.ImageDomain) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateDeleting
}

func wantImageRetryCreate(resource *regional.ImageDomain) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateCreating
}
