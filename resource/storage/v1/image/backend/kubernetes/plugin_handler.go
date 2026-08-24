package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"log"

	"k8s.io/apimachinery/pkg/runtime/schema"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

	frameworkbackend "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
)

// blockStorageGVR is the GroupVersionResource of the block storage an image is stored on.
var blockStorageGVR = schema.GroupVersionResource{Group: bsdom.Group, Version: bsdom.Version, Resource: bsdom.Resource}

// DependencyResolver resolves the block storage an image depends on via its BlockStorageRef.
type DependencyResolver interface {
	State(ctx context.Context, gvr schema.GroupVersionResource, ref commondomain.Reference, defaultTenant string) (bool, commondomain.ResourceState, error)
}

// ImagePluginHandler drives the image reconciliation state machine.
type ImagePluginHandler struct {
	frameworkbackend.GenericPluginHandler[*imgdom.Image]
	repo   persistence.Repo[*imgdom.Image]
	plugin ImagePlugin
	deps   DependencyResolver
}

var _ backendport.PluginHandler[*imgdom.Image] = (*ImagePluginHandler)(nil)

// NewImagePluginHandler creates a new ImagePluginHandler.
func NewImagePluginHandler(
	repo persistence.Repo[*imgdom.Image],
	plugin ImagePlugin,
	maxConditions int,
	deps DependencyResolver,
) *ImagePluginHandler {
	handler := &ImagePluginHandler{
		repo:   repo,
		plugin: plugin,
		deps:   deps,
	}
	handler.MaxConditions = maxConditions

	return handler
}

func (h *ImagePluginHandler) HandleReconcile(ctx context.Context, resource *imgdom.Image) error {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isImageActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
	}

	var delegate backendport.DelegatedFunc[*imgdom.Image]

	switch {

	case isImageAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*imgdom.Image]

	case isImagePending(resource):
		delegate = frameworkbackend.BypassDelegated[*imgdom.Image]

	case isImageCreating(resource):
		delegate = h.plugin.Create

	case wantImageDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*imgdom.Image]

	case isImageDeleting(resource):
		delegate = h.plugin.Delete

	case wantImageRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*imgdom.Image]

	default:
		return nil // Nothing to do.
	}

	if err := delegate(ctx, resource); err != nil {
		var rq backendport.RequeueError
		if errors.As(err, &rq) {
			// Not a failure, and not ours to reinterpret: the plugin named its own cadence.
			return err
		}

		// The failure is recorded on the resource; retry it on the next pass, unless it is
		// already gone, in which case there is nothing left to reconcile.
		return commonbackend.RequeueAfterState(h.setResourceErrorState(ctx, resource, err))
	}

	switch {

	case isImageAccepted(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStatePending))

	case isImagePending(resource):
		return h.ensureBlockStorageReady(ctx, resource)

	case isImageCreating(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStateActive))

	case wantImageDelete(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting))

	case isImageDeleting(resource):
		// Nothing to do: the controller will remove the finalizers to end the deletion process.
		return nil

	case wantImageRetryCreate(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))

	default:
		log.Fatal("must never achieve that condition")
	}

	return nil
}

func (h *ImagePluginHandler) setResourceState(ctx context.Context, resource *imgdom.Image, state commondomain.ResourceState) error {
	if resource.Status == nil {
		resource.Status = &imgdom.ImageStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, err := h.repo.UpdateStatus(ctx, resource)

	return err
}

func (h *ImagePluginHandler) setResourceErrorState(ctx context.Context, resource *imgdom.Image, err error) error {
	if resource.Status == nil {
		resource.Status = &imgdom.ImageStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, updateErr := h.repo.UpdateStatus(ctx, resource)

	return updateErr
}

// ensureBlockStorageReady gates the image's transition to creating on its referenced
// block storage existing and being active. While the dependency is not ready the image
// stays pending and the reconcile is requeued.
func (h *ImagePluginHandler) ensureBlockStorageReady(ctx context.Context, resource *imgdom.Image) error {
	exists, state, err := h.deps.State(ctx, blockStorageGVR, resource.Spec.BlockStorageRef, resource.Tenant)
	if err != nil {
		return commonbackend.RequeueAfterState(h.setResourceErrorState(ctx, resource, err))
	}

	if !exists || state != commondomain.ResourceStateActive {
		message := fmt.Sprintf("waiting for block storage %q to be active", resource.Spec.BlockStorageRef.Resource)
		c := commonbackend.DependencyPendingCondition(commondomain.ResourceStatePending, message)

		return commonbackend.RequeueAfterState(h.setResourceCondition(ctx, resource, c))
	}

	return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
}

// setResourceCondition pushes c onto the resource status and persists it, mirroring
// setResourceState but without forcing a lifecycle transition beyond what c carries.
func (h *ImagePluginHandler) setResourceCondition(ctx context.Context, resource *imgdom.Image, c commondomain.StatusCondition) error {
	if resource.Status == nil {
		resource.Status = &imgdom.ImageStatus{}
	}

	resource.Status.PushCondition(c)
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, err := h.repo.UpdateStatus(ctx, resource)

	return err
}

func isImageActive(resource *imgdom.Image) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
}

func isImageAccepted(resource *imgdom.Image) bool {
	return resource.Status == nil
}

func isImagePending(resource *imgdom.Image) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
}

func isImageCreating(resource *imgdom.Image) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func imageIsNotDeleting(resource *imgdom.Image) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantImageDelete(resource *imgdom.Image) bool {
	return resource.DeletedAt != nil && imageIsNotDeleting(resource)
}

func isImageDeleting(resource *imgdom.Image) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantImageRetryCreate(resource *imgdom.Image) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
