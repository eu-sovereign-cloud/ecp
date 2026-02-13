package controller

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"slices"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

// GenericController implements a generic Kubernetes controller that reconciles
// resources by delegating the logic to a PluginHandler.
//
// It is designed to work with any resource that implements the IdentifiableResource
// interface and has a corresponding Kubernetes representation (CRD).
type GenericController[D gateway.IdentifiableResource] struct {
	client       client.Client
	k8sToDomain  kubernetes.K8sToDomain[D]
	handler      delegator.PluginHandler[D]
	prototype    client.Object
	requeueAfter time.Duration
	logger       *slog.Logger
}

// NewGenericController creates a new instance of GenericController.
func NewGenericController[D gateway.IdentifiableResource](
	client client.Client,
	k8sToDomain kubernetes.K8sToDomain[D],
	handler delegator.PluginHandler[D],
	prototype client.Object,
	requeueAfter time.Duration,
	logger *slog.Logger,
) *GenericController[D] {
	return &GenericController[D]{
		client:       client,
		k8sToDomain:  k8sToDomain,
		handler:      handler,
		prototype:    prototype,
		requeueAfter: requeueAfter,
		logger:       logger,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *GenericController[D]) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(r.prototype).
		Complete(r)
}

const finalizerName = "secapi.cloud.foundation/cleanup"

// Reconcile implements the reconcile.Reconciler interface.
func (r *GenericController[D]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.logger.With("resource", req.NamespacedName)

	log.Println(" --->> 0 - REQ", "req", req)

	var obj client.Object

	// 1. Fetch the K8s object
	obj = r.prototype.DeepCopyObject().(client.Object)
	if err := r.client.Get(ctx, req.NamespacedName, obj); err != nil {
		log.Println(" --->> 1 - ERR", "err", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Println(" --->> 1 - OBJ", "obj", obj)

	// 2. Handle finalizers
	if obj.GetDeletionTimestamp().IsZero() && !slices.Contains(obj.GetFinalizers(), finalizerName) {
		obj.SetFinalizers(append(obj.GetFinalizers(), finalizerName))
		if err := r.client.Update(ctx, obj); err != nil {
			log.Println(" --->> 2 - ERR", "err", err)
			return ctrl.Result{}, err
		}

		log.Println(" --->> 2 - OBJ", "obj", obj)
		return ctrl.Result{RequeueAfter: r.requeueAfter}, nil
	}

	// 3. Convert to Domain object for normal reconciliation
	domainResource, err := r.k8sToDomain(obj)
	if err != nil {
		log.Println(" --->> 3 - ERR", "err", err)
		// If conversion fails, it's likely a permanent error
		logger.Error("failed to convert k8s object to domain resource", "error", err)

		r.updateStatusCondition(ctx, obj, metav1.Condition{
			Type:               "ConversionFailed",
			Status:             metav1.ConditionTrue,
			Reason:             "DomainConversionFailed",
			Message:            err.Error(),
			LastTransitionTime: metav1.Now(),
		})

		log.Println(" --->> 3 - ERROR STATUS SET")
		return ctrl.Result{}, nil
	}

	log.Println(" --->> 3 - DOMAIN", "domainResource", domainResource)

	// 4. Delegate to the specific handler
	requeue, err := r.handler.HandleReconcile(ctx, domainResource)
	if err != nil {
		log.Println(" --->> 4 - ERR", "err", err)

		if errors.Is(err, delegator.ErrStillProcessing) {
			return ctrl.Result{RequeueAfter: r.requeueAfter}, nil
		}
		logger.Error("handler failed to reconcile", "error", err)
		return ctrl.Result{RequeueAfter: r.requeueAfter}, err
	}

	log.Println(" --->> 4 - SUCESS")

	// 5. Requeue the request if necessary
	if requeue {
		log.Println(" --->> 5 - REQUEUE")
		return ctrl.Result{RequeueAfter: r.requeueAfter}, nil
	}

	// 6. Refresh the K8s object
	obj = r.prototype.DeepCopyObject().(client.Object)
	if err := r.client.Get(ctx, req.NamespacedName, obj); err != nil {
		log.Println(" --->> 6 - ERR", "err", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Println(" --->> 6 - OBJ", "obj", obj)

	// 7. Check if the resource deletion process is complete
	if !obj.GetDeletionTimestamp().IsZero() && getStateFromObject(obj) == regional.ResourceStateDeleting && slices.Contains(obj.GetFinalizers(), finalizerName) {
		log.Println(" --->> 7 - REMOVE FINALIZERS")
		obj.SetFinalizers(slices.DeleteFunc(obj.GetFinalizers(), func(v string) bool {
			return strings.EqualFold(v, finalizerName)
		}))
		if err := r.client.Update(ctx, obj); err != nil {
			log.Println(" --->> 7 - ERR", "err", err)
			return ctrl.Result{}, err
		}
	}

	log.Println(" --->> 8 - THE END")
	return ctrl.Result{}, nil
}

func (r *GenericController[D]) updateStatusCondition(ctx context.Context, obj client.Object, condition metav1.Condition) {
	logger := r.logger.With("resource", client.ObjectKeyFromObject(obj))

	// Update status via unstructured
	//
	// TODO: refactor according the issue https://github.com/eu-sovereign-cloud/ecp/issues/180
	// Use an interface to help to manage the conditions instead convert to unstructured.
	uMap, mapErr := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if mapErr != nil {
		logger.Error("failed to convert object to unstructured for status update", "error", mapErr)
		return
	}
	uObj := &unstructured.Unstructured{Object: uMap}

	// Ensure GVK is set (ToUnstructured might not set it if obj TypeMeta is empty)
	if uObj.GetKind() == "" {
		uObj.SetGroupVersionKind(r.prototype.GetObjectKind().GroupVersionKind())
	}

	// Extract existing status conditions from the unstructured object.
	statusConditions := &struct {
		Conditions []metav1.Condition `json:"conditions,omitempty"`
	}{}
	statusMap, found, err := unstructured.NestedMap(uObj.Object, "status")
	if err != nil {
		logger.Error("failed to extract status from unstructured object", "error", err)
		return
	}

	if found {
		// If the status field exists, convert it to our structured type.
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(statusMap, statusConditions); err != nil {
			logger.Error("failed to convert status to structured conditions", "error", err)
			return
		}
	}

	// Use the meta helper to set the new condition. This correctly handles
	// adding a new condition or updating an existing one.
	meta.SetStatusCondition(&statusConditions.Conditions, condition)

	// Convert the updated conditions back to an unstructured map.
	newStatusMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(statusConditions)
	if err != nil {
		logger.Error("failed to convert structured conditions to unstructured", "error", err)
		return
	}

	// Set the updated status back on the object.
	if err := unstructured.SetNestedMap(uObj.Object, newStatusMap, "status"); err != nil {
		logger.Error("failed to set status map in unstructured object", "error", err)
		return
	}

	if err := r.client.Status().Update(ctx, uObj); err != nil {
		logger.Error("failed to update status", "error", err)
	}
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
