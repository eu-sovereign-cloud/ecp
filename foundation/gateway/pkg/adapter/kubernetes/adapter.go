package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kerrs "k8s.io/apimachinery/pkg/api/errors"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation/filter"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

// K8sConverter defines a function that converts a Kubernetes client.Object to a specific type T.
type K8sConverter[T any] func(object client.Object) (T, error)

// DomainConverter defines a function that converts a specific type T to a Kubernetes client.Object.
type DomainConverter[T any] func(domain T) (*unstructured.Unstructured, error)

// Adapter implements the port.ResourceQueryRepository interface for a specific resource type.
type Adapter[T port.NamespacedResource] struct {
	client   dynamic.Interface
	gvr      schema.GroupVersionResource
	logger   *slog.Logger
	toDomain K8sConverter[T]
	toK8s    DomainConverter[T]
}

// Create implements the port.Writer interface.
func (a *Adapter[T]) Create(ctx context.Context, m T) error {
	if a.toK8s == nil {
		return fmt.Errorf("create not supported: missing domain to k8s converter")
	}
	k8sObj, err := a.toK8s(m)
	if err != nil {
		return fmt.Errorf("failed to convert domain object to k8s: %w", err)
	}

	var ri dynamic.ResourceInterface
	nri := a.client.Resource(a.gvr)
	if ns := k8sObj.GetNamespace(); ns != "" {
		ri = nri.Namespace(ns)
	} else {
		ri = nri
	}

	created, err := ri.Create(ctx, k8sObj, metav1.CreateOptions{})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create resource", "resource", a.gvr.Resource, "error", err)
		return fmt.Errorf("failed to create %s: %w", a.gvr.Resource, err)
	}

	converted, err := a.toDomain(created)
	if err != nil {
		return fmt.Errorf("failed to convert created k8s object back to domain: %w", err)
	}

	// Update m with converted
	valM := reflect.ValueOf(m)
	valConverted := reflect.ValueOf(converted)
	if valM.Kind() == reflect.Ptr && valConverted.Kind() == reflect.Ptr {
		valM.Elem().Set(valConverted.Elem())
	}

	return nil
}

// Update implements the port.Writer interface.
func (a *Adapter[T]) Update(ctx context.Context, m T) error {
	if a.toK8s == nil {
		return fmt.Errorf("update not supported: missing domain to k8s converter")
	}
	uobj, err := a.toK8s(m)
	if err != nil {
		return fmt.Errorf("failed to toDomain domain object to k8s: %w", err)
	}

	var ri dynamic.ResourceInterface
	nri := a.client.Resource(a.gvr)
	if ns := uobj.GetNamespace(); ns != "" {
		ri = nri.Namespace(ns)
	} else {
		ri = nri
	}

	updated, err := ri.Update(ctx, uobj, metav1.UpdateOptions{})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to update resource", "resource", a.gvr.Resource, "error", err)
		return fmt.Errorf("failed to update %s: %w", a.gvr.Resource, err)
	}

	converted, err := a.toDomain(updated)
	if err != nil {
		return fmt.Errorf("failed to toDomain updated k8s object back to domain: %w", err)
	}

	// Update m with converted
	valM := reflect.ValueOf(m)
	valConverted := reflect.ValueOf(converted)
	if valM.Kind() == reflect.Ptr && valConverted.Kind() == reflect.Ptr {
		valM.Elem().Set(valConverted.Elem())
	}

	return nil
}

// Delete implements the port.Writer interface.
func (a *Adapter[T]) Delete(ctx context.Context, m T) error {
	var ri dynamic.ResourceInterface
	nri := a.client.Resource(a.gvr)
	if ns := m.GetNamespace(); ns != "" {
		ri = nri.Namespace(ns)
	} else {
		ri = nri
	}

	err := ri.Delete(ctx, m.GetName(), metav1.DeleteOptions{})
	if err != nil {
		if kerrs.IsNotFound(err) {
			return model.ErrNotFound
		}
		a.logger.ErrorContext(ctx, "failed to delete resource", "resource", a.gvr.Resource, "error", err)
		return fmt.Errorf("failed to delete %s: %w", a.gvr.Resource, err)
	}
	return nil
}

// List implements the port.ResourceRepository interface.
func (a *Adapter[T]) List(ctx context.Context, params model.ListParams, list *[]T) (*string, error) {
	lo := metav1.ListOptions{}
	if params.Limit > 0 {
		lo.Limit = int64(params.Limit)
	}
	if params.SkipToken != "" {
		lo.Continue = params.SkipToken
	}

	// Separate server-side and client-side selectors
	if params.Selector != "" {
		lo.LabelSelector = filter.K8sSelectorForAPI(params.Selector)
	}

	var ri dynamic.ResourceInterface = a.client.Resource(a.gvr)
	if params.Namespace != "" {
		ri = a.client.Resource(a.gvr).Namespace(params.Namespace)
	}

	ulist, err := ri.List(ctx, lo)
	modelErr := model.ErrUnavailable
	if kerrs.IsForbidden(err) {
		modelErr = model.ErrForbidden
	}
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to list resources", "resource", a.gvr.Resource, "error", err)
		return nil, fmt.Errorf("%w: failed to list resources for %s: %w", modelErr, a.gvr.Resource, err)

	}

	// Apply client-side filtering for selectors not handled by the API
	var filteredItems []unstructured.Unstructured
	if params.Selector != "" {
		for _, item := range ulist.Items {
			matched, k8sHandled, err := filter.MatchLabels(item.GetLabels(), params.Selector)
			if err != nil {
				a.logger.ErrorContext(ctx, "label filter evaluation failed", "resource", a.gvr.Resource, "item", item.GetName(), "error", err)
				return nil, fmt.Errorf("%w: label filter for %s failed: %w", model.ErrValidation, a.gvr.Resource, err)
			}
			if k8sHandled { // The filter was fully handled by the K8s API
				filteredItems = ulist.Items
				break
			}
			if matched {
				filteredItems = append(filteredItems, item)
			}
		}
	} else {
		filteredItems = ulist.Items
	}

	*list = make([]T, 0, len(filteredItems))
	for _, item := range filteredItems {
		converted, err := a.toDomain(&item)
		if err != nil {
			a.logger.ErrorContext(ctx, "conversion failed", "resource", a.gvr.Resource, "error", err)
			return nil, fmt.Errorf("%w: failed to convert %s: %w", model.ErrValidation, a.gvr.Resource, err)
		}
		*list = append(*list, converted)
	}
	next := ulist.GetContinue()
	if next == "" {
		return nil, nil
	}
	return &next, nil
}

// Load implements the port.ResourceRepository interface.
func (a *Adapter[T]) Load(ctx context.Context, obj *T) error {
	var ri dynamic.ResourceInterface = a.client.Resource(a.gvr)
	v := *obj
	if v.GetNamespace() != "" {
		ri = a.client.Resource(a.gvr).Namespace(v.GetNamespace())
	}
	uobj, err := ri.Get(ctx, v.GetName(), metav1.GetOptions{})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get resource", "name", v.GetNamespace(), "resource", a.gvr.Resource, "error", err)
		modelErr := model.ErrUnavailable
		if kerrs.IsNotFound(err) {
			modelErr = model.ErrNotFound
		}
		return fmt.Errorf("%w: failed to retrieve %s '%s': %w", modelErr, a.gvr.Resource, v.GetName(), err)
	}
	converted, err := a.toDomain(uobj)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion failed", "resource", a.gvr.Resource, "error", err)
		return fmt.Errorf("%w: failed to convert %s: %w", model.ErrValidation, a.gvr.Resource, err)
	}
	*obj = converted
	return nil
}
