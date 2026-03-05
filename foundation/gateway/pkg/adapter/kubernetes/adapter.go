package kubernetes

import (
	"context"
	"crypto/sha3"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation/filter"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
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

// CreateNamespace creates a Kubernetes Namespace.
func CreateNamespace(ctx context.Context, clientSet kubernetes.Interface, name string, ownerLabels map[string]string) (created bool, err error) {
	if name == "" {
		return false, fmt.Errorf("cannot create namespace with empty name")
	}

	if clientSet == nil {
		return false, fmt.Errorf("cannot create namespace %q: clientSet is nil", name)
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: ownerLabels,
		},
	}

	if _, err := clientSet.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
		if kerrs.IsAlreadyExists(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to create namespace %s: %w", name, err)
	}

	return true, nil
}

// DeleteNamespace deletes the namespace. It does not error if NotFound.
func DeleteNamespace(ctx context.Context, clientSet kubernetes.Interface, name string) error {
	if name == "" {
		return nil
	}

	if clientSet == nil {
		return fmt.Errorf("cannot delete namespace %q: clientSet is nil", name)
	}

	if err := clientSet.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if kerrs.IsNotFound(err) {
			return nil
		}

		return err

	}
	return nil
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
	if err != nil {
		modelErr := model.ErrUnavailable
		if kerrs.IsForbidden(err) {
			modelErr = model.ErrForbidden
		}

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
		modelErr := model.ErrUnavailable
		if kerrs.IsNotFound(err) {
			// We need to to return a specific error to signad the "Not Found"
			// condition, but we don't need to log that because it's normal.
			modelErr = model.ErrNotFound
		} else {
			// Otherwise, for other errors, we need to log them.
			a.logger.ErrorContext(ctx, "failed to get resource", "name", v.GetName(), "resource", a.gvr.Resource, "error", err)
		}

		return fmt.Errorf("%w: failed to retrieve %s '%s': %w", modelErr, a.gvr.Resource, v.GetName(), err)
	}

	converted, err := a.k8sToDomain(uobj)
	if err != nil {
		// We even log the conversion errors.
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

	_, err = ri.Create(ctx, uobj, metav1.CreateOptions{})
	if err != nil { // TODO: check if the map error function work for that case
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

	var ures *unstructured.Unstructured
	err = wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		ures, err = ri.Get(ctx, uobj.GetName(), metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		// TODO: simplify this block using proper unstructured methods
		status, found, err := unstructured.NestedMap(ures.Object, "status")
		if err != nil {
			return true, err
		}

		if !found {
			return false, nil
		}

		istate, found := status["state"]
		if !found {
			return false, nil
		}

		state, ok := istate.(string)
		if !ok {
			return false, nil
		}

		if state == "" {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to fetch the k8s resource status", "resource", a.gvr.Resource, "error", err)

		return nil, fmt.Errorf("server error: failed to fetch the %s status: %w", a.gvr.Resource, err) // TODO: review the "server error"
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
	// 1 - Convert from business domain model to K8s unstructured
	uobj, err := a.toUnstructured(m)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion from T to unstructured failed", "resource", a.gvr.Resource, "error", err)

		return nil, fmt.Errorf("%w: failed to convert %s to unstructured: %w", model.ErrValidation, a.gvr.Resource, err)
	}

	// 2 - Decompose the object by extracting...

	// 2.1 - Labels
	desiredLabels := uobj.GetLabels()
	labelsFound := len(desiredLabels) > 0

	// 2.2 - Annotations
	desiredAnnotations := uobj.GetAnnotations()
	annotationsFound := len(desiredAnnotations) > 0

	// 2.3 - The Spec
	// Determine if a status update is intended by checking for meaningful status data.
	desiredSpec, specFound, err := unstructured.NestedMap(uobj.Object, "spec")
	if err != nil {
		return nil, err // TODO: better error handling
	}

	// 2.4 - The status
	desiredStatus, statusFound, err := unstructured.NestedMap(uobj.Object, "status")
	if err != nil {
		return nil, err
	}

	// 3 - Setup the K8s Client Interface to the resource and namespace
	resourceInterface := a.client.Resource(a.gvr).Namespace(ComputeNamespace(m))

	// 4 - Update the Spec if present
	if labelsFound || annotationsFound || specFound {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			currObj, getErr := resourceInterface.Get(ctx, m.GetName(), metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}

			// Skip spec updating when deleting
			if !currObj.GetDeletionTimestamp().IsZero() {
				return nil
			}

			// Extract the current spec
			currSpec, currSpecFound, err := unstructured.NestedMap(currObj.Object, "spec")
			if err != nil {
				return err // TODO: better error handling
			}

			// Set labels
			currObj.SetLabels(desiredLabels)

			// Set annotations
			currObj.SetAnnotations(desiredAnnotations)

			// Set the desired spec on the current object
			if currSpecFound && !cmp.Equal(currSpec, desiredSpec) {
				if err := unstructured.SetNestedMap(currObj.Object, desiredSpec, "spec"); err != nil {
					return err // TODO: better error handling
				}
			}

			// Update the resource spec on K8s
			if _, err := resourceInterface.Update(ctx, currObj, metav1.UpdateOptions{}); err != nil {
				return err // TODO: better error handling
			}

			// At this point everything is OK
			return nil
		})
		if err != nil {
			return nil, a.mapUpdateError(err, m.GetName())
		}
	}

	// 5 - Update the status if present
	if statusFound {
		if err := a.updateStatus(ctx, resourceInterface, m, desiredStatus); err != nil {
			return nil, a.mapUpdateError(err, m.GetName())
		}
	}

	// 6 - Load the final object from K8s
	currObj, err := resourceInterface.Get(ctx, m.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, a.mapUpdateError(err, m.GetName())
	}

	res, err := a.k8sToDomain(currObj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert from k8s object: %w", err)
	}

	return &res, nil
}

func (a *WriterAdapter[T]) updateMetadataAndSpec(
	ctx context.Context,
	ri dynamic.ResourceInterface,
	name string,
	desired *unstructured.Unstructured,
) error {
	desiredLabels := desired.GetLabels()
	desiredAnnotations := desired.GetAnnotations()
	desiredSpec, specFound, err := unstructured.NestedMap(desired.Object, "spec")
	if err != nil {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		currObj, getErr := ri.Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}

		if !currObj.GetDeletionTimestamp().IsZero() {
			return nil
		}

		currSpec, currSpecFound, err := unstructured.NestedMap(currObj.Object, "spec")
		if err != nil {
			return err
		}

		currLabels := currObj.GetLabels()
		currAnnotations := currObj.GetAnnotations()

		specChanged := specFound && (!currSpecFound || !cmp.Equal(currSpec, desiredSpec))
		labelsChanged := !cmp.Equal(currLabels, desiredLabels)
		annotationsChanged := !cmp.Equal(currAnnotations, desiredAnnotations)

		if !specChanged && !labelsChanged && !annotationsChanged {
			return nil
		}

		if specChanged {
			if err := unstructured.SetNestedMap(currObj.Object, desiredSpec, "spec"); err != nil {
				return err
			}
		}
		if labelsChanged {
			currObj.SetLabels(desiredLabels)
		}
		if annotationsChanged {
			currObj.SetAnnotations(desiredAnnotations)
		}

		_, err = ri.Update(ctx, currObj, metav1.UpdateOptions{})

		return err
	})
}

func (a *WriterAdapter[T]) updateStatus(
	ctx context.Context,
	ri dynamic.ResourceInterface,
	m T,
	desiredStatus map[string]interface{},
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		currObj, getErr := ri.Get(ctx, m.GetName(), metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}

		currStatus, found, err := unstructured.NestedMap(currObj.Object, "status")
		if err != nil {
			return err
		}

		if found && cmp.Equal(currStatus, desiredStatus) {
			return nil
		}

		if err := unstructured.SetNestedMap(currObj.Object, desiredStatus, "status"); err != nil {
			return err
		}

		_, err = ri.UpdateStatus(ctx, currObj, metav1.UpdateOptions{})

		return err
	})
}

func (a *WriterAdapter[T]) mapUpdateError(err error, name string) error {
	a.logger.ErrorContext(context.Background(), "failed to update resource", "name", name, "resource", a.gvr.Resource, "error", err)
	var errModel error
	switch {
	case kerrs.IsNotFound(err):
		errModel = model.ErrNotFound
	case kerrs.IsGone(err):
		errModel = model.ErrGone
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
		a.logger.ErrorContext(ctx, "failed to delete resource", "name", m.GetName(), "resource", a.gvr.Resource, "error", err, slog.Any("m", m))

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

// Delete deletes the resource and then attempts to delete the associated namespace only if it is owned by us.
func (a *NamespaceManagingWriterAdapter[T]) Delete(ctx context.Context, m T) error {
	// The current SECA resource organization (https://spec.secapi.cloud/docs/content/Architecture/resource-organization)
	// contains only three hierarchical levels: Tenants 1<->* Workspaces 1<->* SECA Resources.
	//
	// At present, there is not a Tenant entity defined, and the only entity
	// which will really manage its namespace is the Workspace.
	//
	// The Workspace should be placed into the Tenant namespace, and it should
	// not own that namespace because it will contains all the Workspaces and
	// other elements owned by the Tenant.
	//
	// So, in fact, the Workspaces will create and manage namespaces for its
	// underlying resources, and not for themselves.
	//
	// That's why the namespace name must always consider Tenant and Workspace
	// names here.

	// Delete the resource which manages the namespace
	if err := a.WriterAdapter.Delete(ctx, m); err != nil {
		return err
	}

	return nil
}

// Watch implements the port.WatcherRepo interface.
func (a *WatcherAdapter[T]) Watch(ctx context.Context, m chan<- T) error {
	_ = ctx
	_ = m
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

// NamespaceManagingWriterAdapter wraps a WriterAdapter and ensures namespaces exist before creating resources.
// It uses a typed clientset for Namespace operations when available.
type NamespaceManagingWriterAdapter[T port.IdentifiableResource] struct {
	*WriterAdapter[T]
	client    dynamic.Interface
	clientset kubernetes.Interface
	logger    *slog.Logger
}

// RepoAdapter implements the port.WatcherRepo interface for a specific resource type.
type NamespaceManagingRepoAdapter[T port.IdentifiableResource] struct {
	*ReaderAdapter[T]
	*NamespaceManagingWriterAdapter[T]
	*WatcherAdapter[T]
}

// NewNamespaceManagingWriterAdapter creates a new writer adapter that ensures namespaces for resources.
func NewNamespaceManagingWriterAdapter[T port.IdentifiableResource](
	dynClient dynamic.Interface,
	clientset kubernetes.Interface,
	gvr schema.GroupVersionResource,
	logger *slog.Logger,
	domainToK8s DomainToK8s[T],
	k8sToDomain K8sToDomain[T],
) *NamespaceManagingWriterAdapter[T] {
	base := NewWriterAdapter(dynClient, gvr, logger, domainToK8s, k8sToDomain)
	return &NamespaceManagingWriterAdapter[T]{
		WriterAdapter: base,
		client:        dynClient,
		clientset:     clientset,
		logger:        logger,
	}
}

// NewRepoAdapter creates a new Kubernetes adapter for the port.WriterRepo port.
func NewNamespaceManagingRepoAdapter[T port.IdentifiableResource](
	dynClient dynamic.Interface,
	clientset kubernetes.Interface,
	gvr schema.GroupVersionResource,
	logger *slog.Logger,
	domainToK8s DomainToK8s[T],
	k8sToDomain K8sToDomain[T],
) *NamespaceManagingRepoAdapter[T] {
	return &NamespaceManagingRepoAdapter[T]{
		ReaderAdapter: NewReaderAdapter(
			dynClient,
			gvr,
			logger,
			k8sToDomain,
		),
		NamespaceManagingWriterAdapter: NewNamespaceManagingWriterAdapter[T](
			dynClient,
			clientset,
			gvr,
			logger,
			domainToK8s,
			k8sToDomain,
		),
		WatcherAdapter: NewWatcherAdapter(
			dynClient,
			gvr,
			logger,
			k8sToDomain,
		),
	}
}

// namespaceOwnedBy checks that the namespace contains all key/value pairs in expectedLabels.
func namespaceOwnedBy(ctx context.Context, clientset kubernetes.Interface, nsName string, expectedLabels map[string]string) (bool, error) {
	if clientset == nil {
		return false, fmt.Errorf("clientset is nil")
	}

	ns, err := clientset.CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
	if err != nil {
		if kerrs.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	if ns.Labels == nil && len(expectedLabels) > 0 {
		return false, nil
	}

	for k, v := range expectedLabels {
		if got, ok := ns.Labels[k]; !ok || got != v {
			return false, nil
		}
	}

	return true, nil
}

// Create ensures the computed namespace exists (for both tenant and workspace scopes) and rolls back if we created it and the resource creation fails.
func (a *NamespaceManagingWriterAdapter[T]) Create(ctx context.Context, m T) (*T, error) {
	// The current SECA resource organization (https://spec.secapi.cloud/docs/content/Architecture/resource-organization)
	// contains only three hierarchical levels: Tenants 1<->* Workspaces 1<->* SECA Resources.
	//
	// At present, there is not a Tenant entity defined, and the only entity
	// which will really manage its namespace is the Workspace.
	//
	// The Workspace should be placed into the Tenant namespace, and it should
	// not own that namespace because it will contains all the Workspaces and
	// other elements owned by the Tenant.
	//
	// So, in fact, the Workspaces will create and manage namespaces for its
	// underlying resources, and not for themselves.
	//
	// That's why the namespace name must always consider Tenant and Workspace
	// names here.
	tenant := m.GetTenant()
	container := m.GetWorkspace()
	if container == "" {
		container = m.GetName()
	}

	namespace := ComputeNamespace(&scope.Scope{Tenant: tenant, Workspace: container})
	if namespace == "" {
		return a.WriterAdapter.Create(ctx, m)
	}

	ownerLabels := map[string]string{}
	if tenant != "" {
		ownerLabels[labels.InternalTenantLabel] = tenant
	}

	if container != "" {
		ownerLabels[labels.InternalWorkspaceLabel] = container
	}

	createdNS, err := CreateNamespace(ctx, a.clientset, namespace, ownerLabels)
	if err != nil {
		return nil, err
	}

	res, err := a.WriterAdapter.Create(ctx, m)
	if err != nil {
		// rollback namespace only if we created it here and we still own it
		if createdNS {
			if owned, getErr := namespaceOwnedBy(ctx, a.clientset, namespace, ownerLabels); getErr == nil && owned {
				if delErr := DeleteNamespace(ctx, a.clientset, namespace); delErr != nil && !kerrs.IsNotFound(delErr) {
					a.logger.ErrorContext(ctx, "failed to rollback namespace created for resource", "namespace", namespace, "error", delErr)
				}
			} else if getErr != nil {
				a.logger.ErrorContext(ctx, "failed to verify namespace ownership during rollback", "namespace", namespace, "error", getErr)
			}
		}

		return nil, err
	}

	return res, nil
}
