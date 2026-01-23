package kubernetes

import (
	"context"
	"crypto/sha3"
	"errors"
	"fmt"
	"log/slog"

	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation/filter"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

// K8sToDomain defines a function that converts a Kubernetes client.Object to a specific type T.
type K8sToDomain[T any] func(object client.Object) (T, error)

// DomainToK8s defines a function that converts a domain type T to a Kubernetes client.Object.
type DomainToK8s[T any] func(domain T) (client.Object, error)

// Adapter is the base struct for Kubernetes adapters.
type Adapter struct {
	client dynamic.Interface
	gvr    schema.GroupVersionResource
	logger *slog.Logger
}

// ReaderAdapter implements the port.ReaderRepo interface for a specific resource type.
type ReaderAdapter[T port.IdentifiableResource] struct {
	Adapter
	k8sToDomain K8sToDomain[T]
}

// WriterAdapter implements the port.WriterRepo interface for a specific resource type.
type WriterAdapter[T port.IdentifiableResource] struct {
	Adapter
	domainToK8s DomainToK8s[T]
	k8sToDomain K8sToDomain[T]
}

// WatcherAdapter implements the port.WatcherRepo interface for a specific resource type.
type WatcherAdapter[T port.IdentifiableResource] struct {
	Adapter
	k8sToDomain K8sToDomain[T]
}

// RepoAdapter implements the port.WatcherRepo interface for a specific resource type.
type RepoAdapter[T port.IdentifiableResource] struct {
	*ReaderAdapter[T]
	*WriterAdapter[T]
	*WatcherAdapter[T]
}

// NewRepoAdapter creates a new Kubernetes adapter for the port.WriterRepo port.
func NewRepoAdapter[T port.IdentifiableResource](
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	logger *slog.Logger,
	domainToK8s DomainToK8s[T],
	k8sToDomain K8sToDomain[T],
) *RepoAdapter[T] {
	return &RepoAdapter[T]{
		ReaderAdapter: NewReaderAdapter(
			client,
			gvr,
			logger,
			k8sToDomain,
		),
		WriterAdapter: NewWriterAdapter(
			client,
			gvr,
			logger,
			domainToK8s,
			k8sToDomain,
		),
		WatcherAdapter: NewWatcherAdapter(
			client,
			gvr,
			logger,
			k8sToDomain,
		),
	}
}

// NewReaderAdapter creates a new Kubernetes adapter for the port.ReaderRepo port.
func NewReaderAdapter[T port.IdentifiableResource](
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	logger *slog.Logger,
	k8sToDomain K8sToDomain[T],
) *ReaderAdapter[T] {
	return &ReaderAdapter[T]{
		Adapter: Adapter{
			client: client,
			gvr:    gvr,
			logger: logger,
		},
		k8sToDomain: k8sToDomain,
	}
}

// NewWriterAdapter creates a new Kubernetes adapter for the port.WriterRepo port.
func NewWriterAdapter[T port.IdentifiableResource](
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	logger *slog.Logger,
	domainToK8s DomainToK8s[T],
	k8sToDomain K8sToDomain[T],
) *WriterAdapter[T] {
	return &WriterAdapter[T]{
		Adapter: Adapter{
			client: client,
			gvr:    gvr,
			logger: logger,
		},
		domainToK8s: domainToK8s,
		k8sToDomain: k8sToDomain,
	}
}

// NewWatcherAdapter creates a new Kubernetes adapter for the port.ReaderRepo port.
func NewWatcherAdapter[T port.IdentifiableResource](
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	logger *slog.Logger,
	k8sToDomain K8sToDomain[T],
) *WatcherAdapter[T] {
	return &WatcherAdapter[T]{
		Adapter: Adapter{
			client: client,
			gvr:    gvr,
			logger: logger,
		},
		k8sToDomain: k8sToDomain,
	}
}

// ComputeNamespace computes the Kubernetes namespace based on tenant and workspace.
func ComputeNamespace(obj port.Scope) string {
	if obj.GetTenant() == "" && obj.GetWorkspace() == "" {
		return ""
	}

	hasher := sha3.New224()
	if obj.GetTenant() != "" && obj.GetWorkspace() == "" {
		_, _ = fmt.Fprintf(hasher, "%s", obj.GetTenant())
	} else {
		_, _ = fmt.Fprintf(hasher, "%s/%s", obj.GetTenant(), obj.GetWorkspace())
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// List implements the port.ReaderRepo interface.
func (a *ReaderAdapter[T]) List(ctx context.Context, params model.ListParams, list *[]T) (*string, error) {
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

	ri := a.client.Resource(a.gvr).Namespace(ComputeNamespace(&params))

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

// Load implements the port.ReaderRepo interface.
func (a *ReaderAdapter[T]) Load(ctx context.Context, obj *T) error {
	v := *obj
	ri := a.client.Resource(a.gvr).Namespace(ComputeNamespace(v))

	uobj, err := ri.Get(ctx, v.GetName(), metav1.GetOptions{})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get resource", "name", v.GetName(), "resource", a.gvr.Resource, "error", err)
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

// Create implements the port.WriterRepo interface.
func (a *WriterAdapter[T]) Create(ctx context.Context, m T) (*T, error) {
	ri := a.client.Resource(a.gvr).Namespace(ComputeNamespace(m))

	uobj, err := a.toUnstructured(m)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion to k8s object failed", "resource", a.gvr.Resource, "error", err)
		return nil, fmt.Errorf("%w: failed to convert %s to k8s object: %w", model.ErrValidation, a.gvr.Resource, err)
	}

	ures, err := ri.Create(ctx, uobj, metav1.CreateOptions{})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create resource", "name", m.GetName(), "resource", a.gvr.Resource, "error", err)

		var errModel error
		switch {
		case kerrs.IsNotFound(err): // occurs when the namespace of the resource does not exist
			errModel = model.ErrNotFound
		case kerrs.IsAlreadyExists(err): // occurs when the resource with the same name already exists
			errModel = model.ErrAlreadyExists
		case kerrs.IsInvalid(err): // occurs when the resource is semantically invalid
			errModel = model.ErrValidation
		default:
			errModel = model.ErrUnavailable
		}

		return nil, fmt.Errorf("%w: failed to create %s '%s': %w", errModel, a.gvr.Resource, m.GetName(), err)
	}

	res, err := a.k8sToDomain(ures)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion from k8s object failed", "resource", a.gvr.Resource, "error", err)
		return nil, fmt.Errorf("%w: failed to convert %s from k8s object: %w", model.ErrValidation, a.gvr.Resource, err)
	}

	return &res, nil
}

// Update implements the port.WriterRepo interface. It handles both spec-only updates
// and updates that include the status, using the appropriate Kubernetes API endpoints.
func (a *WriterAdapter[T]) Update(ctx context.Context, m T) (*T, error) {
	ri := a.client.Resource(a.gvr).Namespace(ComputeNamespace(m))

	uobj, err := a.toUnstructured(m)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion from T to unstructured failed", "resource", a.gvr.Resource, "error", err)
		return nil, fmt.Errorf("%w: failed to convert %s to unstructured: %w", model.ErrValidation, a.gvr.Resource, err)
	}

	// Determine if a status update is intended by checking for meaningful status data.
	desiredStatus, statusExists, _ := unstructured.NestedMap(uobj.Object, "status")
	conditions, conditionsExist, _ := unstructured.NestedSlice(desiredStatus, "conditions")

	if !statusExists || !conditionsExist || len(conditions) == 0 {
		// --- PATH A: Spec-only update (or status without conditions) ---
		var ures *unstructured.Unstructured
		updateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			currentU, getErr := ri.Get(ctx, m.GetName(), metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}
			uobj.SetResourceVersion(currentU.GetResourceVersion())

			currentStatus, found, _ := unstructured.NestedMap(currentU.Object, "status")
			if !found {
				currentStatus = make(map[string]interface{})
			}
			if _, condsFound := currentStatus["conditions"]; !condsFound {
				currentStatus["conditions"] = []interface{}{}
			}
			if err := unstructured.SetNestedField(uobj.Object, currentStatus, "status"); err != nil {
				return err // This should not happen, but return if it does.
			}

			var updateErr error
			ures, updateErr = ri.Update(ctx, uobj, metav1.UpdateOptions{})
			return updateErr
		})

		if updateErr != nil {
			return nil, a.mapUpdateError(updateErr, m.GetName())
		}
		res, err := a.k8sToDomain(ures)
		if err != nil {
			return nil, fmt.Errorf("failed to convert from k8s object: %w", err)
		}
		return &res, nil
	}

	// --- PATH B: Update includes status with conditions ---
	specUpdateObj := uobj.DeepCopy()
	unstructured.RemoveNestedField(specUpdateObj.Object, "status")

	updateSpecErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		currentObj, getErr := ri.Get(ctx, m.GetName(), metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		specUpdateObj.SetResourceVersion(currentObj.GetResourceVersion())
		_, updateErr := ri.Update(ctx, specUpdateObj, metav1.UpdateOptions{})
		return updateErr
	})
	if updateSpecErr != nil {
		return nil, a.mapUpdateError(updateSpecErr, m.GetName())
	}

	statusUpdateObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": uobj.GetAPIVersion(),
			"kind":       uobj.GetKind(),
			"metadata":   map[string]interface{}{"name": uobj.GetName(), "namespace": uobj.GetNamespace()},
			"status":     desiredStatus,
		},
	}

	updateStatusErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		currentObj, getErr := ri.Get(ctx, m.GetName(), metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		statusUpdateObj.SetResourceVersion(currentObj.GetResourceVersion())
		_, updateErr := ri.UpdateStatus(ctx, statusUpdateObj, metav1.UpdateOptions{})
		return updateErr
	})
	if updateStatusErr != nil {
		return nil, a.mapUpdateError(updateStatusErr, m.GetName())
	}

	finalUobj, getErr := ri.Get(ctx, m.GetName(), metav1.GetOptions{})
	if getErr != nil {
		return nil, fmt.Errorf("failed to retrieve updated resource: %w", getErr)
	}
	res, err := a.k8sToDomain(finalUobj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert from k8s object: %w", err)
	}
	return &res, nil
}

func (a *WriterAdapter[T]) mapUpdateError(err error, name string) error {
	a.logger.ErrorContext(context.Background(), "failed to update resource", "name", name, "resource", a.gvr.Resource, "error", err)
	var errModel error
	switch {
	case kerrs.IsNotFound(err):
		errModel = model.ErrNotFound
	case kerrs.IsConflict(err):
		errModel = model.ErrConflict
	case kerrs.IsInvalid(err):
		errModel = model.ErrValidation
	default:
		errModel = model.ErrUnavailable
	}
	return fmt.Errorf("%w: failed to update %s '%s': %w", errModel, a.gvr.Resource, name, err)
}

// Delete implements the port.WriterRepo interface.
func (a *WriterAdapter[T]) Delete(ctx context.Context, m T) error {
	ri := a.client.Resource(a.gvr).Namespace(ComputeNamespace(m))

	deleteOptions := metav1.DeleteOptions{}
	if m.GetVersion() != "" {
		deleteOptions.Preconditions = &metav1.Preconditions{
			ResourceVersion: ptr.To(m.GetVersion()),
		}
	}
	err := ri.Delete(ctx, m.GetName(), deleteOptions)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to delete resource", "name", m.GetName(), "resource", a.gvr.Resource, "error", err)

		if kerrs.IsNotFound(err) {
			return fmt.Errorf("%w: %s '%s' not found", model.ErrNotFound, a.gvr.Resource, m.GetName())
		}
		if kerrs.IsInvalid(err) {
			return fmt.Errorf("%w: %s '%s': %w", model.ErrValidation, a.gvr.Resource, m.GetName(), err)
		}
		if kerrs.IsConflict(err) {
			return fmt.Errorf("%w: %s '%s': %w", model.ErrConflict, a.gvr.Resource, m.GetName(), err)
		}
		return fmt.Errorf("%w: failed to delete %s '%s': %w", model.ErrUnavailable, a.gvr.Resource, m.GetName(), err)
	}

	return nil
}

// Watch implements the port.WatcherRepo interface.
func (a *WatcherAdapter[T]) Watch(ctx context.Context, m chan<- T) error {
	// TODO: implement the watch method of the kubernetes repo adapter.
	return errors.New("not implemented")
}

func (a *WriterAdapter[T]) toUnstructured(m T) (*unstructured.Unstructured, error) {
	obj, err := a.domainToK8s(m)
	if err != nil {
		a.logger.Error("conversion to k8s object failed", "resource", a.gvr.Resource, "error", err)
		return nil, fmt.Errorf("failed to convert %s to k8s object: %w", a.gvr.Resource, err)
	}

	uobj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		a.logger.Error("conversion to unstructured failed", "resource", a.gvr.Resource, "error", err)
		return nil, fmt.Errorf("failed to convert k8s object to unstructured: %w", err)
	}

	return &unstructured.Unstructured{Object: uobj}, nil
}
