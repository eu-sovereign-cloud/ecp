package plugin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpconditions "github.com/crossplane/crossplane-runtime/v2/pkg/conditions"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

type base struct {
	client client.Client
	logger *slog.Logger
}

// reconcileError returns a non-nil error if the provider has reported a reconcile
// error on the Synced condition, or nil while still provisioning.
func reconcileError(obj xpconditions.ObjectWithConditions) error {
	if synced := obj.GetCondition(v1.TypeSynced); synced.Equal(v1.ReconcileError(errors.New(synced.Message))) {
		return fmt.Errorf("provider failed to reconcile %s: %s", obj.GetObjectKind().GroupVersionKind().Kind, synced.Message)
	}
	return nil
}

// checkExisting fetches obj and evaluates its readiness.
// Returns nil when ready, ErrStillProcessing while provisioning, or an error on failure.
func (b *base) checkExisting(
	ctx context.Context,
	obj xpconditions.ObjectWithConditions,
) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	err := b.client.Get(ctx, client.ObjectKeyFromObject(obj), obj)
	if err != nil {
		b.logger.Error("failed to check "+kind+" existence", "name", obj.GetName(), "namespace", obj.GetNamespace(), "error", err)
		return err
	}
	if err := reconcileError(obj); err != nil {
		b.logger.Error(kind+" in error state", "name", obj.GetName(), "namespace", obj.GetNamespace(), "error", err)
		return err
	}
	if obj.GetCondition(v1.TypeReady).Status == corev1.ConditionTrue {
		b.logger.Info(kind+" is ready", "name", obj.GetName(), "namespace", obj.GetNamespace())
		return nil
	}
	b.logger.Info(kind+" not yet ready, waiting", "name", obj.GetName(), "namespace", obj.GetNamespace())
	return delegator.ErrStillProcessing
}
