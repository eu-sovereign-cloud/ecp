// foundation/gateway/internal/provider/kubernetes/adapter.go
package kubernetes

import (
	"context"
	"fmt"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/port"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation/filter"
)

// UnstructuredConverter remains a useful helper within the adapter.
type UnstructuredConverter[T any] func(unstructured.Unstructured) (T, error)

// Adapter implements the port.ResourceQueryRepository interface for a specific resource type.
type Adapter[T any] struct {
	client  dynamic.Interface
	gvr     schema.GroupVersionResource
	logger  *slog.Logger
	convert UnstructuredConverter[T]
}

// NewAdapter creates a new Kubernetes adapter for the port.ResourceQueryRepository port.
func NewAdapter[T any](
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	logger *slog.Logger,
	convert UnstructuredConverter[T],
) *Adapter[T] {
	return &Adapter[T]{
		client:  client,
		gvr:     gvr,
		logger:  logger,
		convert: convert,
	}
}

// List implements the port.ResourceRepository interface.
func (a *Adapter[T]) List(ctx context.Context, params port.ListParams) ([]T, *string, error) {
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
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to list resources", "resource", a.gvr.Resource, "error", err)
		return nil, nil, fmt.Errorf("failed to list resources for %s: %w", a.gvr.Resource, err)
	}

	// Apply client-side filtering for selectors not handled by the API
	var filteredItems []unstructured.Unstructured
	if params.Selector != "" {
		for _, item := range ulist.Items {
			matched, k8sHandled, err := filter.MatchLabels(item.GetLabels(), params.Selector)
			if err != nil {
				a.logger.ErrorContext(ctx, "label filter evaluation failed", "resource", a.gvr.Resource, "item", item.GetName(), "error", err)
				return nil, nil, fmt.Errorf("label filter for %s failed: %w", a.gvr.Resource, err)
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

	result := make([]T, 0, len(filteredItems))
	for _, item := range filteredItems {
		converted, err := a.convert(item)
		if err != nil {
			a.logger.ErrorContext(ctx, "conversion failed", "resource", a.gvr.Resource, "error", err)
			return nil, nil, fmt.Errorf("failed to convert %s: %w", a.gvr.Resource, err)
		}
		result = append(result, converted)
	}

	next := ulist.GetContinue()
	if next == "" {
		return result, nil, nil
	}
	return result, &next, nil
}

// Get implements the port.ResourceRepository interface.
func (a *Adapter[T]) Get(ctx context.Context, namespace, name string) (T, error) {
	// This method's body would be a refactored version of the original `GetResource` function.
	var zero T
	var ri dynamic.ResourceInterface = a.client.Resource(a.gvr)
	if namespace != "" {
		ri = a.client.Resource(a.gvr).Namespace(namespace)
	}

	uobj, err := ri.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get resource", "name", name, "resource", a.gvr.Resource, "error", err)
		return zero, fmt.Errorf("failed to retrieve %s '%s': %w", a.gvr.Resource, name, err)
	}

	return a.convert(*uobj)
}

// DefaultUnstructuredConverter can remain a helper in this package.
func DefaultUnstructuredConverter[T any]() UnstructuredConverter[T] {
	return func(obj unstructured.Unstructured) (T, error) {
		var out T
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &out); err != nil {
			var zero T
			return zero, fmt.Errorf("default converter failed: %w", err)
		}
		return out, nil
	}
}
