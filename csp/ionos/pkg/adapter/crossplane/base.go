package crossplane

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpconditions "github.com/crossplane/crossplane-runtime/v2/pkg/conditions"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	backend "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
)

const (
	ProviderConfigName = "cluster-ionos-provider-config"
	ProviderConfigType = "ClusterProviderConfig"
)

type base struct {
	client client.Client
	logger *slog.Logger
}

func (c *base) createCR(ctx context.Context, obj xpconditions.ObjectWithConditions) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if err := c.client.Create(ctx, obj); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			c.logger.Error("failed to create "+kind, "name", obj.GetName(), "error", err)
			return err
		}
		return c.checkExisting(ctx, obj)
	}
	c.logger.Info(kind+" created, waiting for ready", "name", obj.GetName())
	return backend.StillProcessing
}

func (c *base) updateCR(ctx context.Context, obj xpconditions.ObjectWithConditions) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if err := c.client.Update(ctx, obj); err != nil {
		c.logger.Error("failed to update "+kind, "name", obj.GetName(), "error", err)
		return err
	}
	c.logger.Info(kind+" updated, waiting for ready", "name", obj.GetName())
	return backend.StillProcessing
}

func (c *base) deleteCR(ctx context.Context, obj xpconditions.ObjectWithConditions) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if err := c.client.Delete(ctx, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		c.logger.Error("failed to delete "+kind, "name", obj.GetName(), "error", err)
		return err
	}
	if err := c.client.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		c.logger.Error("failed to check "+kind+" deletion state", "name", obj.GetName(), "error", err)
		return err
	}
	if err := reconcileError(obj); err != nil {
		c.logger.Error(kind+" deletion failed", "name", obj.GetName(), "error", err)
		return err
	}
	c.logger.Info("waiting for "+kind+" deletion", "name", obj.GetName())
	return backend.StillProcessing
}

func (c *base) checkExisting(ctx context.Context, obj xpconditions.ObjectWithConditions) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if err := c.client.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		c.logger.Error("failed to check "+kind+" existence", "name", obj.GetName(), "error", err)
		return err
	}
	if err := reconcileError(obj); err != nil {
		c.logger.Error(kind+" in error state", "name", obj.GetName(), "error", err)
		return err
	}

	if obj.GetDeletionTimestamp() != nil {
		c.logger.Info(kind+" is being deleted", "name", obj.GetName())
		return backend.StillProcessing
	}

	readyCond := obj.GetCondition(v1.TypeReady)
	generationSeen := readyCond.ObservedGeneration == 0 || readyCond.ObservedGeneration == obj.GetGeneration()
	if readyCond.Status == corev1.ConditionTrue && generationSeen {
		c.logger.Info(kind+" is ready", "name", obj.GetName())
		return nil
	}
	c.logger.Info(kind+" not yet ready", "name", obj.GetName())
	return backend.StillProcessing
}

// reconcileError reports the provider's own reconcile failure, if the Crossplane managed resource
// is carrying one.
//
// The cause arrives as a string on the Synced condition — Crossplane serialises it over the wire,
// so there is no original error value to unwrap. It is turned back into an error and wrapped in a
// kernel.Error rather than flattened into a message, so a caller can errors.As it like every other
// failure crossing this boundary; the provider being unable to converge is an upstream outage from
// here, hence KindUnavailable.
func reconcileError(obj xpconditions.ObjectWithConditions) error {
	synced := obj.GetCondition(v1.TypeSynced)
	cause := errors.New(synced.Message)
	if !synced.Equal(v1.ReconcileError(cause)) {
		return nil
	}

	kind := obj.GetObjectKind().GroupVersionKind().Kind
	return kernel.NewError(kernel.KindUnavailable,
		fmt.Errorf("provider failed to reconcile %s: %w", kind, cause),
		kernel.ErrorSource{Name: kind, Value: obj.GetName()})
}
