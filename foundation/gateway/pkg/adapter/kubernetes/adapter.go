package kubernetes

import (
	"context"
	"crypto/sha3"
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
	kerrs "k8s.io/apimachinery/pkg/api/errors"
)

// K8sConverter defines a function that converts a Kubernetes client.Object to a specific type T.
type K8sConverter[T any] func(object client.Object) (T, error)

// Adapter implements the port.ResourceQueryRepository interface for a specific resource type.
type Adapter[T port.IdentifiableResource] struct {
	client  dynamic.Interface
	gvr     schema.GroupVersionResource
	logger  *slog.Logger
	convert K8sConverter[T]
}

// NewAdapter creates a new Kubernetes adapter for the port.ResourceQueryRepository port.
func NewAdapter[T port.IdentifiableResource](
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

// computeNamespace computes the Kubernetes namespace based on tenant and workspace.
func computeNamespace(obj port.Scope) string {
	if obj.GetTenant() == "" && obj.GetWorkspace() == "" {
		return ""
	}

	val := sha3.New224()
	if obj.GetTenant() != "" && obj.GetWorkspace() == "" {
		_, _ = fmt.Fprintf(val, "%s", obj.GetTenant())
	} else {
		_, _ = fmt.Fprintf(val, "%s/%s", obj.GetTenant(), obj.GetWorkspace())
	}

	return fmt.Sprintf("%x", val.Sum(nil))
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
	ri = a.client.Resource(a.gvr).Namespace(computeNamespace(&params))

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
		converted, err := a.convert(&item)
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
	ri = a.client.Resource(a.gvr).Namespace(computeNamespace(v))

	uobj, err := ri.Get(ctx, v.GetName(), metav1.GetOptions{})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get resource", "name", v.GetName(), "resource", a.gvr.Resource, "error", err)
		modelErr := model.ErrUnavailable
		if kerrs.IsNotFound(err) {
			modelErr = model.ErrNotFound
		}
		return fmt.Errorf("%w: failed to retrieve %s '%s': %w", modelErr, a.gvr.Resource, v.GetName(), err)
	}
	converted, err := a.convert(uobj)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion failed", "resource", a.gvr.Resource, "error", err)
		return fmt.Errorf("%w: failed to convert %s: %w", model.ErrValidation, a.gvr.Resource, err)
	}
	*obj = converted
	return nil
}
