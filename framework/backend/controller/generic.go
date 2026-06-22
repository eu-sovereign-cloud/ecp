package controller

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes"
	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"

	backend "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
)

// stateDeleting is the wire value of ResourceState when a resource is being deleted.
// It matches the "deleting" sentinel stored in the status.state field of every CRD.
// Using a local const severs the framework's dependency on any resources package.
// If the sentinel ever changes, update this const and its test assertion.
const stateDeleting = "deleting"

// GenericController implements a generic Kubernetes controller that reconciles
// resources by delegating the logic to a PluginHandler.
//
// It is designed to work with any resource that implements the IdentifiableResource
// interface and has a corresponding Kubernetes representation (CRD).
type GenericController[D persistence.IdentifiableResource] struct {
	client              client.Client
	k8sToDomain         k8sadapter.K8sToDomain[D]
	handler             backend.PluginHandler[D]
	prototype           schemav1.ConditionedObject
	requeueAfter        time.Duration
	logger              *slog.Logger
	maxStatusConditions int
}

// NewGenericController creates a new instance of GenericController.
func NewGenericController[D persistence.IdentifiableResource](
	client client.Client,
	k8sToDomain k8sadapter.K8sToDomain[D],
	handler backend.PluginHandler[D],
	prototype schemav1.ConditionedObject,
	requeueAfter time.Duration,
	logger *slog.Logger,
	maxStatusConditions int,
) GenericController[D] {
	return GenericController[D]{
		client:              client,
		k8sToDomain:         k8sToDomain,
		handler:             handler,
		prototype:           prototype,
		requeueAfter:        requeueAfter,
		logger:              logger,
		maxStatusConditions: maxStatusConditions,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *GenericController[D]) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(r.prototype).
		WithOptions(controller.Options{
			// This allows 10 workers to process the queue in parallel
			// TODO: make this configurable
			MaxConcurrentReconciles: 10,
		}).
		Complete(r)
}

const finalizerName = "secapi.cloud.foundation/cleanup"

// Reconcile implements the reconcile.Reconciler interface.
func (r *GenericController[D]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.logger.With("resource", req.NamespacedName)

	var obj schemav1.ConditionedObject

	// 1. Fetch the K8s object
	obj = r.prototype.DeepCopyObject().(schemav1.ConditionedObject)
	if err := r.client.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Handle finalizers
	if obj.GetDeletionTimestamp().IsZero() && !slices.Contains(obj.GetFinalizers(), finalizerName) {
		obj.SetFinalizers(append(obj.GetFinalizers(), finalizerName))
		if err := r.client.Update(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: r.requeueAfter}, nil
	}

	// 3. Convert to Domain object for normal reconciliation
	domainResource, err := r.k8sToDomain(obj)
	if err != nil {
		// If conversion fails, it's likely a permanent error
		logger.Error("failed to convert k8s object to domain resource", "error", err)

		obj.PushCondition(schemav1.StatusCondition{
			State:            schemav1.ResourceStateError,
			Type:             "ConversionFailed",
			Reason:           "DomainConversionFailed",
			Message:          err.Error(),
			LastTransitionAt: metav1.Now(),
		})

		for r.maxStatusConditions > 0 && obj.LenConditions() > r.maxStatusConditions {
			obj.PopCondition()
		}

		if err = r.client.Status().Update(ctx, obj); err != nil {
			logger.Error("failed to update status", "error", err)
		}
		return ctrl.Result{}, nil
	}

	// 4. Delegate to the specific handler
	requeue, err := r.handler.HandleReconcile(ctx, domainResource)
	if err != nil {
		if errors.Is(err, backend.ErrStillProcessing) {
			return ctrl.Result{RequeueAfter: r.requeueAfter}, nil
		}
		logger.Error("handler failed to reconcile", "error", err)
		return ctrl.Result{RequeueAfter: r.requeueAfter}, err
	}

	// 5. Requeue the request if necessary
	if requeue {
		return ctrl.Result{RequeueAfter: r.requeueAfter}, nil
	}

	// 6. Refresh the K8s object
	obj = r.prototype.DeepCopyObject().(schemav1.ConditionedObject)
	if err := r.client.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 7. Check if the resource deletion process is complete
	if !obj.GetDeletionTimestamp().IsZero() &&
		getStateFromObject(obj) == stateDeleting &&
		slices.Contains(obj.GetFinalizers(), finalizerName) {
		obj.SetFinalizers(slices.DeleteFunc(obj.GetFinalizers(), func(v string) bool {
			return strings.EqualFold(v, finalizerName)
		}))
		if err := r.client.Update(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// getStateFromObject reads the status.state field from any ConditionedObject via
// unstructured conversion. Returns the raw string value or "" on error.
func getStateFromObject(obj client.Object) string {
	// Extract status via unstructured
	uMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return ""
	}
	uObj := &unstructured.Unstructured{Object: uMap}

	state, found, err := unstructured.NestedString(uObj.Object, "status", "state")
	if err != nil || !found {
		return ""
	}

	return state
}
