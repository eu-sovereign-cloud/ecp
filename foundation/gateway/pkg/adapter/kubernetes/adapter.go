package kubernetes

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation/filter"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
)

// K8sToDomain defines a function that converts a Kubernetes client.Object to a specific domain type T.
type K8sToDomain[T any] func(object client.Object) (T, error)

// DomainToK8s defines a function that converts a domain type T into a Kubernetes client.Object.
type DomainToK8s[T any] func(obj T) (client.Object, error)

// Adapter implements the port.ResourceQueryRepository interface for a specific resource type.
type Adapter[T port.NamespacedResource] struct {
	client      dynamic.Interface
	gvr         schema.GroupVersionResource
	logger      *slog.Logger
	k8sToDomain K8sToDomain[T]
	domainToK8s DomainToK8s[T]
}

// NewAdapter creates a new Kubernetes adapter for the port.ResourceQueryRepository port.
func NewAdapter[T port.NamespacedResource](
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	logger *slog.Logger,
	k8sToDomain K8sToDomain[T],
	domainToK8s DomainToK8s[T],
) *Adapter[T] {
	if k8sToDomain == nil || domainToK8s == nil {
		log.Fatalf("failed to create adapter: resource converter is nil %s", gvr.Resource)
	}

	return &Adapter[T]{
		client:      client,
		gvr:         gvr,
		logger:      logger,
		k8sToDomain: k8sToDomain,
		domainToK8s: domainToK8s,
	}
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
		converted, err := a.k8sToDomain(&item)
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
	converted, err := a.k8sToDomain(uobj)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion failed", "resource", a.gvr.Resource, "error", err)
		return fmt.Errorf("%w: failed to convert %s: %w", model.ErrValidation, a.gvr.Resource, err)
	}
	*obj = converted
	return nil
}

// Create implements the port.Writer interface.
func (a *Adapter[T]) Create(ctx context.Context, obj T) error {
	uobj, err := a.convertToUnstructured(ctx, obj)
	if err != nil {
		return err
	}

	var ri dynamic.ResourceInterface = a.client.Resource(a.gvr)
	if obj.GetNamespace() != "" {
		ri = a.client.Resource(a.gvr).Namespace(obj.GetNamespace())
	}

	_, err = ri.Create(ctx, uobj, metav1.CreateOptions{})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create resource", "name", obj.GetName(), "resource", a.gvr.Resource, "error", err)
		modelErr := model.ErrUnavailable
		switch {
		case kerrs.IsAlreadyExists(err):
			modelErr = model.ErrConflict
		case kerrs.IsForbidden(err):
			modelErr = model.ErrForbidden
		case kerrs.IsInvalid(err):
			modelErr = model.ErrValidation
		}
		return fmt.Errorf("%w: failed to create %s '%s': %w", modelErr, a.gvr.Resource, obj.GetName(), err)
	}

	return nil
}

// Update implements the port.Writer interface.
func (a *Adapter[T]) Update(ctx context.Context, obj T) error {
	uobj, err := a.convertToUnstructured(ctx, obj)
	if err != nil {
		return err
	}

	var ri dynamic.ResourceInterface = a.client.Resource(a.gvr)
	if obj.GetNamespace() != "" {
		ri = a.client.Resource(a.gvr).Namespace(obj.GetNamespace())
	}

	_, err = ri.Update(ctx, uobj, metav1.UpdateOptions{})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to update resource", "name", obj.GetName(), "resource", a.gvr.Resource, "error", err)
		modelErr := model.ErrUnavailable
		switch {
		case kerrs.IsNotFound(err):
			modelErr = model.ErrNotFound
		case kerrs.IsForbidden(err):
			modelErr = model.ErrForbidden
		case kerrs.IsConflict(err):
			modelErr = model.ErrConflict
		case kerrs.IsInvalid(err):
			modelErr = model.ErrValidation
		}
		return fmt.Errorf("%w: failed to update %s '%s': %w", modelErr, a.gvr.Resource, obj.GetName(), err)
	}

	return nil
}

// Delete implements the port.Writer interface.
func (a *Adapter[T]) Delete(ctx context.Context, obj T) error {
	var ri dynamic.ResourceInterface = a.client.Resource(a.gvr)
	if obj.GetNamespace() != "" {
		ri = a.client.Resource(a.gvr).Namespace(obj.GetNamespace())
	}

	err := ri.Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to delete resource", "name", obj.GetName(), "resource", a.gvr.Resource, "error", err)
		modelErr := model.ErrUnavailable
		if kerrs.IsNotFound(err) {
			modelErr = model.ErrNotFound
		} else if kerrs.IsForbidden(err) {
			modelErr = model.ErrForbidden
		}
		return fmt.Errorf("%w: failed to delete %s '%s': %w", modelErr, a.gvr.Resource, obj.GetName(), err)
	}

	return nil
}

// convertToUnstructured uses the provided domainToK8s converter to transform the domain object into
// a Kubernetes client.Object, then into an *unstructured.Unstructured suitable for the dynamic client.
func (a *Adapter[T]) convertToUnstructured(ctx context.Context, obj T) (*unstructured.Unstructured, error) {

	co, err := a.domainToK8s(obj)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to convert domain to k8s object", "resource", a.gvr.Resource, "error", err)
		return nil, fmt.Errorf("%w: failed to map %s: %w", model.ErrValidation, a.gvr.Resource, err)
	}
	if u, ok := co.(*unstructured.Unstructured); ok {
		return u, nil
	}

	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(co)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to convert k8s object to unstructured", "resource", a.gvr.Resource, "error", err)
		return nil, fmt.Errorf("%w: failed to convert %s to unstructured: %w", model.ErrValidation, a.gvr.Resource, err)
	}
	return &unstructured.Unstructured{Object: content}, nil
}
