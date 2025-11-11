package main

import (
	"context"
	"fmt"
	"slices"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	blockstorage "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/block-storage"
	storagev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/block-storage/storages/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/plugin"
)

// SetupStorageController wires the controller into the manager.
func SetupStorageController(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1.Storage{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(&StorageReconciler{client: mgr.GetClient(), scheme: mgr.GetScheme()})
}

type StorageReconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func (r *StorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var obj storagev1.Storage
	logger.Info("Reconciling Storage", "name", req.NamespacedName) // Add this line
	if err := r.client.Get(ctx, req.NamespacedName, &obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	finalizer := "delegator.secapi.cloud/finalizer"

	if obj.DeletionTimestamp.IsZero() {
		if !slices.Contains(obj.Finalizers, finalizer) {
			obj.Finalizers = append(obj.Finalizers, finalizer)
			if err := r.client.Update(ctx, &obj); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if slices.Contains(obj.Finalizers, finalizer) {
			for name, plug := range plugin.Registry {
				if supportsStorage(plug) {
					if err := plug.Delete(ctx, &obj); err != nil {
						return ctrl.Result{}, err
					}
					upsertCondition(&obj, genv1.StatusCondition{Type: ptrStr(fmt.Sprintf("Provider:%s", name)), State: genv1.ResourceStateDeleting, Reason: ptrStr("Deleted")})
				}
			}
			obj.Finalizers = slices.DeleteFunc(obj.Finalizers, func(s string) bool { return s == finalizer })
			if err := r.client.Update(ctx, &obj); err != nil {
				return ctrl.Result{}, err
			}
			_ = r.updateStatus(ctx, &obj)
		}
		return ctrl.Result{}, nil
	}

	if obj.Annotations == nil {
		obj.Annotations = map[string]string{}
	}

	// Validation
	if obj.Spec.SizeGB <= 0 {
		upsertCondition(&obj, genv1.StatusCondition{Type: ptrStr("Validated"), State: genv1.ResourceStateError, Reason: ptrStr("InvalidSpec"), Message: ptrStr("sizeGB must be > 0")})
		_ = r.updateStatus(ctx, &obj)
		return ctrl.Result{}, nil
	}
	upsertCondition(&obj, genv1.StatusCondition{Type: ptrStr("Validated"), State: genv1.ResourceStateActive, Reason: ptrStr("OK")})

	// Call plugins
	allReady := true
	var maxRequeue time.Duration
	for name, plug := range plugin.Registry {
		if !supportsStorage(plug) {
			continue
		}
		condType := fmt.Sprintf("Provider:%s", name)
		res, err := plug.Reconcile(ctx, &obj)
		if err != nil {
			logger.Error(err, "plugin reconcile failed", "plugin", name)
			state := genv1.ResourceStateError
			upsertCondition(&obj, genv1.StatusCondition{Type: ptrStr(condType), State: state, Reason: ptrStr("Error"), Message: ptrStr(err.Error())})
			allReady = false
			continue
		}
		// Map PluginResult state
		var st genv1.ResourceState
		switch res.State {
		case "InProgress", "Pending":
			st = genv1.ResourceStatePending
		case "Succeeded":
			st = genv1.ResourceStateActive
		case "Failed":
			st = genv1.ResourceStateError
		default:
			st = genv1.ResourceStatePending
		}
		upsertCondition(&obj, genv1.StatusCondition{Type: ptrStr(condType), State: st, Reason: ptrStr(res.State), Message: ptrStr(res.Message)})
		if st != genv1.ResourceStateActive {
			allReady = false
		}
		if res.RequeueAfter > 0 && res.RequeueAfter > maxRequeue {
			maxRequeue = res.RequeueAfter
		}
	}

	if allReady {
		upsertCondition(&obj, genv1.StatusCondition{Type: ptrStr("Ready"), State: genv1.ResourceStateActive, Reason: ptrStr("AllProvidersSucceeded")})
	} else {
		upsertCondition(&obj, genv1.StatusCondition{Type: ptrStr("Ready"), State: genv1.ResourceStatePending, Reason: ptrStr("InProgress")})
	}

	if err := r.client.Update(ctx, &obj); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.updateStatus(ctx, &obj); err != nil {
		return ctrl.Result{}, err
	}
	if !allReady || maxRequeue > 0 {
		return ctrl.Result{RequeueAfter: maxDuration(maxRequeue, 2*time.Second)}, nil
	}
	return ctrl.Result{}, nil
}

func upsertCondition(s *storagev1.Storage, c genv1.StatusCondition) {
	c.LastTransitionAt = metav1.Now()
	for i := range s.Status.Conditions {
		if s.Status.Conditions[i].Type != nil && c.Type != nil && *s.Status.Conditions[i].Type == *c.Type {
			if s.Status.Conditions[i].State != c.State {
				c.LastTransitionAt = metav1.Now()
			} else {
				c.LastTransitionAt = s.Status.Conditions[i].LastTransitionAt
			}
			s.Status.Conditions[i] = c
			return
		}
	}
	s.Status.Conditions = append(s.Status.Conditions, c)
}

func findCondition(s *storagev1.Storage, condType string) *genv1.StatusCondition {
	for i := range s.Status.Conditions {
		if s.Status.Conditions[i].Type != nil && *s.Status.Conditions[i].Type == condType {
			return &s.Status.Conditions[i]
		}
	}
	return nil
}

func ptrStr(s string) *string {
	return &s
}

// AddToScheme registers existing regional API types.
func AddToScheme(s *runtime.Scheme) error {
	if err := blockstorage.AddToScheme(s); err != nil {
		return err
	}
	return nil
}

func (r *StorageReconciler) patchStatusCondition(ctx context.Context, nn types.NamespacedName, cond metav1.Condition) error {
	return nil
}

func (r *StorageReconciler) updateStatus(ctx context.Context, s *storagev1.Storage) error {
	return r.client.Status().Update(ctx, s)
}

func supportsStorage(p plugin.ResourcePlugin) bool {
	for _, k := range p.SupportedKinds() {
		if k == storagev1.StorageGVR.String() {
			return true
		}
	}
	return false
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
