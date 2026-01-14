package controller

import (
	"context"
	"log/slog"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

// GenericController implements a generic Kubernetes controller that reconciles
// resources by delegating the logic to a PluginHandler.
//
// It is designed to work with any resource that implements the IdentifiableResource
// interface and has a corresponding Kubernetes representation (CRD).
type GenericController[D gateway.IdentifiableResource] struct {
	client      client.Client
	k8sToDomain kubernetes.K8sToDomain[D]
	handler     delegator.PluginHandler[D]
	prototype   client.Object
	logger      *slog.Logger
}

// NewGenericController creates a new instance of GenericController.
func NewGenericController[D gateway.IdentifiableResource](
	client client.Client,
	k8sToDomain kubernetes.K8sToDomain[D],
	handler delegator.PluginHandler[D],
	prototype client.Object,
	logger *slog.Logger,
) *GenericController[D] {
	return &GenericController[D]{
		client:      client,
		k8sToDomain: k8sToDomain,
		handler:     handler,
		prototype:   prototype,
		logger:      logger,
	}
}

// Reconcile implements the reconcile.Reconciler interface.
func (r *GenericController[D]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.logger.With("resource", req.NamespacedName)

	// 1. Fetch the K8s object
	// We use DeepCopyObject to create a new instance of the prototype to ensure
	// we have a clean object to unmarshal into.
	// K is constrained by client.Object, so the assertion is safe at runtime
	// as long as the prototype is a pointer to a struct that implements client.Object.
	obj := r.prototype.DeepCopyObject().(client.Object)
	if err := r.client.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Convert to Domain object
	domainResource, err := r.k8sToDomain(obj)
	if err != nil {
		// If conversion fails, it's likely a permanent error (e.g. invalid data in CRD
		// that passed schema validation but failed domain validation).
		// We log it and update the status, but do not requeue to avoid hot loops.
		logger.Error("failed to convert k8s object to domain resource", "error", err)

		r.updateStatusCondition(ctx, obj, metav1.Condition{
			Type:               "ConversionFailed",
			Status:             metav1.ConditionTrue,
			Reason:             "DomainConversionFailed",
			Message:            err.Error(),
			LastTransitionTime: metav1.Now(),
		})

		return ctrl.Result{}, nil
	}

	// 3. Delegate to the specific handler
	if err := r.handler.HandleReconcile(ctx, domainResource); err != nil {
		logger.Error("handler failed to reconcile", "error", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GenericController[D]) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(r.prototype).
		Complete(r)
}

func (r *GenericController[D]) updateStatusCondition(ctx context.Context, obj client.Object, condition metav1.Condition) {
	logger := r.logger.With("resource", client.ObjectKeyFromObject(obj))

	// Update status via unstructured
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

	// Update conditions in the unstructured object
	newConditions := []interface{}{}
	existingConditions, found, _ := unstructured.NestedSlice(uObj.Object, "status", "conditions")
	if found {
		for _, c := range existingConditions {
			cMap, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			if cMap["type"] == condition.Type {
				continue
			}
			newConditions = append(newConditions, c)
		}
	}

	cMap := map[string]interface{}{
		"type":               condition.Type,
		"status":             string(condition.Status),
		"reason":             condition.Reason,
		"message":            condition.Message,
		"lastTransitionTime": condition.LastTransitionTime.Format(time.RFC3339),
	}
	newConditions = append(newConditions, cMap)

	if err := unstructured.SetNestedSlice(uObj.Object, newConditions, "status", "conditions"); err != nil {
		logger.Error("failed to set conditions in unstructured object", "error", err)
		return
	}

	if err := r.client.Status().Update(ctx, uObj); err != nil {
		logger.Error("failed to update status", "error", err)
	}
}
