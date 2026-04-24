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

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/generated/types"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/regional/common"

	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

// GenericController implements a generic Kubernetes controller that reconciles
// resources by delegating the logic to a PluginHandler.
//
// It is designed to work with any resource that implements the IdentifiableResource
// interface and has a corresponding Kubernetes representation (CRD).
type GenericController[D gateway.IdentifiableResource] struct {
	client              client.Client
	k8sToDomain         kubernetes.K8sToDomain[D]
	handler             delegator.PluginHandler[D]
	prototype           client.Object
	requeueAfter        time.Duration
	logger              *slog.Logger
	maxStatusConditions int
}

// NewGenericController creates a new instance of GenericController.
func NewGenericController[D gateway.IdentifiableResource](
	client client.Client,
	k8sToDomain kubernetes.K8sToDomain[D],
	handler delegator.PluginHandler[D],
	prototype client.Object,
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
			// This allows 10 wrokers to process the queue in parallel
			// TODO: make this configurable
			MaxConcurrentReconciles: 10,
		}).
		Complete(r)
}

const finalizerName = "secapi.cloud.foundation/cleanup"

// Reconcile implements the reconcile.Reconciler interface.
func (r *GenericController[D]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.logger.With("resource", req.NamespacedName)

	var obj common.ConditionedObject

	// 1. Fetch the K8s object
	obj = r.prototype.DeepCopyObject().(common.ConditionedObject)
	if err := r.client.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("attempting to retrieve status conditions", "conditions", obj.GetStatusConditions())

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

		obj.PushStatusCondition(genv1.StatusCondition{
			State:            genv1.ResourceStateError,
			Type:             "ConversionFailed",
			Reason:           "DomainConversionFailed",
			Message:          err.Error(),
			LastTransitionAt: metav1.Now(),
		})

		if obj.LenStatusConditions() > r.maxStatusConditions {
			obj.PopStatusCondition()
		}

		if err = r.client.Status().Update(ctx, obj); err != nil {
			logger.Error("failed to update status", "error", err)
		}
		return ctrl.Result{}, nil
	}

	// 4. Delegate to the specific handler
	requeue, err := r.handler.HandleReconcile(ctx, domainResource)
	if err != nil {
		if errors.Is(err, delegator.ErrStillProcessing) {
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
	obj = r.prototype.DeepCopyObject().(common.ConditionedObject)
	if err := r.client.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 7. Check if the resource deletion process is complete
	if !obj.GetDeletionTimestamp().IsZero() &&
		getStateFromObject(obj) == regional.ResourceStateDeleting &&
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

func getStateFromObject(obj client.Object) regional.ResourceStateDomain {
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

	return regional.ResourceStateDomain(state)
}
