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

	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

const (
	providerConfigName = "cluster-ionos-provider-config"
	providerConfigType = "ClusterProviderConfig"
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
	return delegator.ErrStillProcessing
}

func (c *base) updateCR(ctx context.Context, obj xpconditions.ObjectWithConditions) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if err := c.client.Update(ctx, obj); err != nil {
		c.logger.Error("failed to update "+kind, "name", obj.GetName(), "error", err)
		return err
	}
	c.logger.Info(kind+" updated, waiting for ready", "name", obj.GetName())
	return delegator.ErrStillProcessing
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
	return delegator.ErrStillProcessing
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
	readyCond := obj.GetCondition(v1.TypeReady)
	generationSeen := readyCond.ObservedGeneration == 0 || readyCond.ObservedGeneration == obj.GetGeneration()
	if readyCond.Status == corev1.ConditionTrue && generationSeen {
		c.logger.Info(kind+" is ready", "name", obj.GetName())
		return nil
	}
	c.logger.Info(kind+" not yet ready", "name", obj.GetName())
	return delegator.ErrStillProcessing
}

func reconcileError(obj xpconditions.ObjectWithConditions) error {
	synced := obj.GetCondition(v1.TypeSynced)
	if synced.Equal(v1.ReconcileError(errors.New(synced.Message))) {
		return fmt.Errorf("provider failed to reconcile %s: %s", obj.GetObjectKind().GroupVersionKind().Kind, synced.Message)
	}
	return nil
}
