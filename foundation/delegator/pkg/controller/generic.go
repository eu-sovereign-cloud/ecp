package controller

import (
	"context"
	"log/slog"

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
type GenericController[D gateway.IdentifiableResource, K client.Object] struct {
	client      client.Client
	k8sToDomain kubernetes.K8sToDomain[D]
	handler     delegator.PluginHandler[D]
	prototype   K
	logger      *slog.Logger
}

// NewGenericController creates a new instance of GenericController.
func NewGenericController[D gateway.IdentifiableResource, K client.Object](
	client client.Client,
	k8sToDomain kubernetes.K8sToDomain[D],
	handler delegator.PluginHandler[D],
	prototype K,
	logger *slog.Logger,
) *GenericController[D, K] {
	return &GenericController[D, K]{
		client:      client,
		k8sToDomain: k8sToDomain,
		handler:     handler,
		prototype:   prototype,
		logger:      logger,
	}
}

// Reconcile implements the reconcile.Reconciler interface.
func (r *GenericController[D, K]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.logger.With("resource", req.NamespacedName)

	// 1. Fetch the K8s object
	// We use DeepCopyObject to create a new instance of the prototype to ensure
	// we have a clean object to unmarshal into.
	// K is constrained by client.Object, so the assertion is safe at runtime
	// as long as the prototype is a pointer to a struct that implements client.Object.
	obj := r.prototype.DeepCopyObject().(K)
	if err := r.client.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Convert to Domain object
	domainResource, err := r.k8sToDomain(obj)
	if err != nil {
		// If conversion fails, it's likely a permanent error (e.g. invalid data in CRD
		// that passed schema validation but failed domain validation).
		// We log it and do not requeue to avoid hot loops.
		logger.Error("failed to convert k8s object to domain resource", "error", err)
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
func (r *GenericController[D, K]) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(r.prototype).
		Complete(r)
}
