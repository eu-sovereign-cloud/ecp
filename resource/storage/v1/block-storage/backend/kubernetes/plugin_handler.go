package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	"k8s.io/apimachinery/pkg/runtime/schema"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

	frameworkbackend "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
)

// imageGVR is the GroupVersionResource of images that may be stored on a block storage.
var imageGVR = schema.GroupVersionResource{Group: imgdom.Group, Version: imgdom.Version, Resource: imgdom.Resource}

// DependencyResolver resolves the optional source image a block storage is created from
// and finds images that are stored on a block storage (referencing it via BlockStorageRef).
type DependencyResolver interface {
	State(ctx context.Context, gvr schema.GroupVersionResource, ref commondomain.Reference, defaultTenant string) (bool, commondomain.ResourceState, error)
	Referrers(ctx context.Context, gvr schema.GroupVersionResource, namespace string, fieldPath []string, target commonbackend.ReferenceTarget, defaultTenant string) ([]string, error)
}

// BlockStoragePluginHandler drives the block-storage reconciliation state machine.
type BlockStoragePluginHandler struct {
	frameworkbackend.GenericPluginHandler[*bsdom.BlockStorage]
	repo   persistence.Repo[*bsdom.BlockStorage]
	plugin BlockStoragePlugin
	deps   DependencyResolver
}

var _ backendport.PluginHandler[*bsdom.BlockStorage] = (*BlockStoragePluginHandler)(nil)

// NewBlockStoragePluginHandler creates a new BlockStoragePluginHandler.
func NewBlockStoragePluginHandler(
	repo persistence.Repo[*bsdom.BlockStorage],
	plugin BlockStoragePlugin,
	maxConditions int,
	deps DependencyResolver,
) *BlockStoragePluginHandler {
	handler := &BlockStoragePluginHandler{
		repo:   repo,
		plugin: plugin,
		deps:   deps,
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
		return h.ensureSourceImageReady(ctx, resource)

	case isBlockStorageCreating(resource):
		resource.Status.SizeGB = resource.Spec.SizeGB

		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)

	case wantBlockStorageDelete(resource):
		return h.ensureNoImageReferrers(ctx, resource)

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

func (h *BlockStoragePluginHandler) setResourceState(ctx context.Context, resource *bsdom.BlockStorage, state commondomain.ResourceState, requeue bool) (bool, error) {
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

// ensureSourceImageReady gates the block storage's transition to creating on its optional
// source image existing and being active. A block storage without a source image proceeds
// immediately.
func (h *BlockStoragePluginHandler) ensureSourceImageReady(ctx context.Context, resource *bsdom.BlockStorage) (bool, error) {
	if resource.Spec.SourceImageRef == nil {
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	}

	exists, state, err := h.deps.State(ctx, imageGVR, *resource.Spec.SourceImageRef, resource.Tenant)
	if err != nil {
		return h.setResourceErrorState(ctx, resource, err, true)
	}

	if !exists || state != commondomain.ResourceStateActive {
		message := fmt.Sprintf("waiting for source image %q to be active", resource.Spec.SourceImageRef.Resource)
		c := commonbackend.DependencyPendingCondition(commondomain.ResourceStatePending, message)
		return h.setResourceCondition(ctx, resource, c, true)
	}

	return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
}

// ensureNoImageReferrers blocks deletion of a block storage while any image is still
// stored on it (references it via BlockStorageRef). The block storage keeps its current
// state (and therefore its cleanup finalizer) until those images are gone. Images are
// tenant-scoped, so referrers are searched in the block storage's tenant namespace.
func (h *BlockStoragePluginHandler) ensureNoImageReferrers(ctx context.Context, resource *bsdom.BlockStorage) (bool, error) {
	namespace := frameworkbackend.ComputeNamespace(&kernelresource.Scope{Tenant: resource.Tenant})
	target := commonbackend.ReferenceTarget{Tenant: resource.Tenant, Workspace: resource.Workspace, Name: resource.Name}

	referrers, err := h.deps.Referrers(ctx, imageGVR, namespace, []string{"spec", "blockStorageRef"}, target, resource.Tenant)
	if err != nil {
		return h.setResourceErrorState(ctx, resource, err, true)
	}

	if len(referrers) > 0 {
		message := fmt.Sprintf("deletion blocked: still referenced by images %v", referrers)
		c := commonbackend.DeletionBlockedCondition(resource.Status.State, message)
		return h.setResourceCondition(ctx, resource, c, true)
	}

	return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)
}

// setResourceCondition pushes c onto the resource status and persists it, mirroring
// setResourceState but without forcing a lifecycle transition beyond what c carries.
func (h *BlockStoragePluginHandler) setResourceCondition(ctx context.Context, resource *bsdom.BlockStorage, c commondomain.StatusCondition, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &bsdom.BlockStorageStatus{}
	}

	resource.Status.PushCondition(c)
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
