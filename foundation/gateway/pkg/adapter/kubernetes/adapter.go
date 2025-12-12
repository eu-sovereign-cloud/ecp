package kubernetes

import (
	"context"
	"fmt"
	"log/slog"

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

// K8sConverter defines a function that converts a Kubernetes Client.Object to a specific type T.
type K8sConverter[T any] func(object client.Object) (T, error)

// DomainToUnstructured defines a function that converts a domain object to unstructured.
type DomainToUnstructured[T any] func(domain T) (*unstructured.Unstructured, error)

// Adapter implements the port.ResourceQueryRepository interface for a specific resource type.
type Adapter[T port.NamespacedResource] struct {
	Client       dynamic.Interface
	GVR          schema.GroupVersionResource
	Logger       *slog.Logger
	K8sConverter K8sConverter[T]
	DomainToK8s  DomainToUnstructured[T]
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

	// Separate server-side and Client-side selectors
	if params.Selector != "" {
		lo.LabelSelector = filter.K8sSelectorForAPI(params.Selector)
	}

	var ri dynamic.ResourceInterface = a.Client.Resource(a.GVR)
	if params.Namespace != "" {
		ri = a.Client.Resource(a.GVR).Namespace(params.Namespace)
	}

	ulist, err := ri.List(ctx, lo)
	modelErr := model.ErrUnavailable
	if kerrs.IsForbidden(err) {
		modelErr = model.ErrForbidden
	}
	if err != nil {
		a.Logger.ErrorContext(ctx, "failed to list resources", "resource", a.GVR.Resource, "error", err)
		return nil, fmt.Errorf("%w: failed to list resources for %s: %w", modelErr, a.GVR.Resource, err)

	}

	// Apply Client-side filtering for selectors not handled by the API
	var filteredItems []unstructured.Unstructured
	if params.Selector != "" {
		for _, item := range ulist.Items {
			matched, k8sHandled, err := filter.MatchLabels(item.GetLabels(), params.Selector)
			if err != nil {
				a.Logger.ErrorContext(ctx, "label filter evaluation failed", "resource", a.GVR.Resource, "item", item.GetName(), "error", err)
				return nil, fmt.Errorf("%w: label filter for %s failed: %w", model.ErrValidation, a.GVR.Resource, err)
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
		converted, err := a.K8sConverter(&item)
		if err != nil {
			a.Logger.ErrorContext(ctx, "conversion failed", "resource", a.GVR.Resource, "error", err)
			return nil, fmt.Errorf("%w: failed to K8sConverter %s: %w", model.ErrValidation, a.GVR.Resource, err)
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
	var ri dynamic.ResourceInterface = a.Client.Resource(a.GVR)
	v := *obj
	if v.GetNamespace() != "" {
		ri = a.Client.Resource(a.GVR).Namespace(v.GetNamespace())
	}
	uobj, err := ri.Get(ctx, v.GetName(), metav1.GetOptions{})
	if err != nil {
		a.Logger.ErrorContext(ctx, "failed to get resource", "name", v.GetNamespace(), "resource", a.GVR.Resource, "error", err)
		modelErr := model.ErrUnavailable
		if kerrs.IsNotFound(err) {
			modelErr = model.ErrNotFound
		}
		return fmt.Errorf("%w: failed to retrieve %s '%s': %w", modelErr, a.GVR.Resource, v.GetName(), err)
	}
	converted, err := a.K8sConverter(uobj)
	if err != nil {
		a.Logger.ErrorContext(ctx, "conversion failed", "resource", a.GVR.Resource, "error", err)
		return fmt.Errorf("%w: failed to K8sConverter %s: %w", model.ErrValidation, a.GVR.Resource, err)
	}
	*obj = converted
	return nil
}

// Create implements the port.Writer interface.
func (a *Adapter[T]) Create(ctx context.Context, obj T) error {
	var ri dynamic.ResourceInterface = a.Client.Resource(a.GVR)
	if obj.GetNamespace() != "" {
		ri = a.Client.Resource(a.GVR).Namespace(obj.GetNamespace())
	}

	// Convert the domain object to unstructured
	unstructuredObj, err := a.toUnstructured(obj)
	if err != nil {
		a.Logger.ErrorContext(ctx, "failed to K8sConverter to unstructured", "gvr", a.GVR.String(), "error", err)
		return fmt.Errorf("failed to K8sConverter %s to unstructured: %w", a.GVR.Resource, err)
	}

	_, err = ri.Create(ctx, unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		a.Logger.ErrorContext(ctx, "failed to create resource", "name", obj.GetName(), "namespace", obj.GetNamespace(), "gvr", a.GVR.String(), "error", err)
		return fmt.Errorf("failed to create %s '%s': %w", a.GVR.Resource, obj.GetName(), err)
	}

	return nil
}

// Update implements the port.Writer interface.
func (a *Adapter[T]) Update(ctx context.Context, obj T) error {
	var ri dynamic.ResourceInterface = a.Client.Resource(a.GVR)
	if obj.GetNamespace() != "" {
		ri = a.Client.Resource(a.GVR).Namespace(obj.GetNamespace())
	}

	// Convert the domain object to unstructured
	unstructuredObj, err := a.toUnstructured(obj)
	if err != nil {
		a.Logger.ErrorContext(ctx, "failed to K8sConverter to unstructured", "gvr", a.GVR.String(), "error", err)
		return fmt.Errorf("failed to K8sConverter %s to unstructured: %w", a.GVR.Resource, err)
	}

	_, err = ri.Update(ctx, unstructuredObj, metav1.UpdateOptions{})
	if err != nil {
		a.Logger.ErrorContext(ctx, "failed to update resource", "name", obj.GetName(), "namespace", obj.GetNamespace(), "gvr", a.GVR.String(), "error", err)
		return fmt.Errorf("failed to update %s '%s': %w", a.GVR.Resource, obj.GetName(), err)
	}

	return nil
}

// Delete implements the port.Writer interface.
func (a *Adapter[T]) Delete(ctx context.Context, obj T) error {
	var ri dynamic.ResourceInterface = a.Client.Resource(a.GVR)
	if obj.GetNamespace() != "" {
		ri = a.Client.Resource(a.GVR).Namespace(obj.GetNamespace())
	}

	err := ri.Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
	if err != nil {
		a.Logger.ErrorContext(ctx, "failed to delete resource", "name", obj.GetName(), "namespace", obj.GetNamespace(), "resource", a.GVR.Resource, "error", err)
		return fmt.Errorf("failed to delete %s '%s': %w", a.GVR.Resource, obj.GetName(), err)
	}

	return nil
}

// Watch implements the port.Watcher interface (stub for now).
func (a *Adapter[T]) Watch(ctx context.Context, m chan<- T) error {
	// TODO: Implement watch functionality
	return fmt.Errorf("watch not implemented")
}

// toUnstructured converts a domain object to an unstructured object for K8s operations.
func (a *Adapter[T]) toUnstructured(obj T) (*unstructured.Unstructured, error) {
	if a.DomainToK8s != nil {
		return a.DomainToK8s(obj)
	}
	// Fallback: simple approach
	u := &unstructured.Unstructured{}
	u.SetName(obj.GetName())
	u.SetNamespace(obj.GetNamespace())
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   a.GVR.Group,
		Version: a.GVR.Version,
		Kind:    a.GVR.Resource,
	})
	return u, nil
}
