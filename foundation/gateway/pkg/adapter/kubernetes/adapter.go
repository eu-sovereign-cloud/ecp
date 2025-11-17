package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation/filter"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

var adapterErr = errors.New("kubernetes adapter error")

// K8sConverter defines a function that converts a Kubernetes client.Object to a specific type T.
type K8sConverter[T any] func(object client.Object) (T, error)

// Adapter implements the port.ResourceQueryRepository interface for a specific resource type.
type Adapter[T port.NamespacedResource] struct {
	client  dynamic.Interface
	gvr     schema.GroupVersionResource
	logger  *slog.Logger
	convert K8sConverter[T]
}

// NewAdapter creates a new Kubernetes adapter for the port.ResourceQueryRepository port.
func NewAdapter[T port.NamespacedResource](
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	logger *slog.Logger,
	convert K8sConverter[T],
) *Adapter[T] {
	return &Adapter[T]{
		client:  client,
		gvr:     gvr,
		logger:  logger,
		convert: convert,
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
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to list resources", "resource", a.gvr.Resource, "error", err)
		return nil, errors.Join(adapterErr, fmt.Errorf("failed to list resources for %s: %w", a.gvr.Resource, err))
	}

	// Apply client-side filtering for selectors not handled by the API
	var filteredItems []unstructured.Unstructured
	if params.Selector != "" {
		for _, item := range ulist.Items {
			matched, k8sHandled, err := filter.MatchLabels(item.GetLabels(), params.Selector)
			if err != nil {
				a.logger.ErrorContext(ctx, "label filter evaluation failed", "resource", a.gvr.Resource, "item", item.GetName(), "error", err)
				return nil, fmt.Errorf("label filter for %s failed: %w", a.gvr.Resource, err)
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
		converted, err := a.convert(&item)
		if err != nil {
			a.logger.ErrorContext(ctx, "conversion failed", "resource", a.gvr.Resource, "error", err)
			return nil, fmt.Errorf("failed to convert %s: %w", a.gvr.Resource, err)
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
		return fmt.Errorf("failed to retrieve %s '%s': %w", a.gvr.Resource, v.GetName(), err)
	}
	converted, err := a.convert(uobj)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion failed", "resource", a.gvr.Resource, "error", err)
		return fmt.Errorf("failed to convert %s: %w", a.gvr.Resource, err)
	}
	*obj = converted
	return nil
}
