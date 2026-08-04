package kubernetes

import (
	"context"
	"crypto/sha3"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation/filter"
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

// ReaderAdapter implements the persistence.ReaderRepo interface for a specific resource type.
type ReaderAdapter[T persistence.IdentifiableResource] struct {
	Adapter
	k8sToDomain K8sToDomain[T]
}

// WriterAdapter implements the persistence.WriterRepo interface for a specific resource type.
type WriterAdapter[T persistence.IdentifiableResource] struct {
	Adapter
	domainToK8s DomainToK8s[T]
	k8sToDomain K8sToDomain[T]
}

// WatcherAdapter implements the persistence.WatcherRepo interface for a specific resource type.
type WatcherAdapter[T persistence.IdentifiableResource] struct {
	Adapter
	k8sToDomain K8sToDomain[T]
}

// RepoAdapter implements the persistence.WatcherRepo interface for a specific resource type.
type RepoAdapter[T persistence.IdentifiableResource] struct {
	*ReaderAdapter[T]
	*WriterAdapter[T]
	*WatcherAdapter[T]
}

// NewRepoAdapter creates a new Kubernetes adapter for the persistence.WriterRepo port.
func NewRepoAdapter[T persistence.IdentifiableResource](
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

// NewReaderAdapter creates a new Kubernetes adapter for the persistence.ReaderRepo port.
func NewReaderAdapter[T persistence.IdentifiableResource](
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

// NewWriterAdapter creates a new Kubernetes adapter for the persistence.WriterRepo port.
func NewWriterAdapter[T persistence.IdentifiableResource](
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

// NewWatcherAdapter creates a new Kubernetes adapter for the persistence.ReaderRepo port.
func NewWatcherAdapter[T persistence.IdentifiableResource](
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

// ComputeNamespace computes the Kubernetes namespace based on tenant and workspace. It never
// looks at anything beyond persistence.Scope — for resolving the right namespace rule for a
// given resource (which may be network-scoped instead), see resolveNamespace.
func ComputeNamespace(obj persistence.Scope) string {
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

// ComputeNetworkNamespace computes the Kubernetes namespace for a network-scoped resource,
// hashing tenant/workspace/network so each network gets its own namespace. This gives
// network-scoped resources (e.g. RouteTable name uniqueness) isolation per network, unlike
// ComputeNamespace's per-workspace sharing used by every other (non-network-scoped) resource.
func ComputeNetworkNamespace(obj persistence.NetworkScope) string {
	hasher := sha3.New224()
	_, _ = fmt.Fprintf(hasher, "%s/%s/%s", obj.GetTenant(), obj.GetWorkspace(), obj.GetNetwork())

	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// resolveNamespace picks the right namespace rule for obj: ComputeNetworkNamespace when obj
// also implements NetworkScope, ComputeNamespace otherwise. This is where "which formula applies
// to this resource" is decided — ComputeNamespace and ComputeNetworkNamespace themselves stay
// dumb, explicit hash formulas with no branching on the caller's shape. A NetworkScope object
// with an empty network is a caller bug, not a fallback case, so it errors instead of silently
// resolving to the workspace-level namespace.
func resolveNamespace(obj persistence.Scope) (string, error) {
	if networkScope, ok := obj.(persistence.NetworkScope); ok {
		if networkScope.GetNetwork() == "" {
			return "", kernel.NewError(kernel.KindValidation, fmt.Errorf("network-scoped resource has empty network"))
		}
		return ComputeNetworkNamespace(networkScope), nil
	}
	return ComputeNamespace(obj), nil
}

// CreateNamespace creates a Kubernetes Namespace.
func CreateNamespace(ctx context.Context, clientSet kubernetes.Interface, name string, ownerLabels map[string]string) (created bool, err error) {
	if name == "" {
		return false, kernel.NewError(kernel.KindValidation, fmt.Errorf("cannot create namespace with empty name"))
	}

	if clientSet == nil {
		return false, kernel.NewError(kernel.KindUnavailable, fmt.Errorf("cannot create namespace %q: clientSet is nil", name))
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

		return false, kubeToDomainError(fmt.Errorf("failed to create namespace %s: %w", name, err))
	}

	return true, nil
}

// DeleteNamespace deletes the namespace. It does not error if NotFound.
func DeleteNamespace(ctx context.Context, clientSet kubernetes.Interface, name string) error {
	if name == "" {
		return nil
	}

	if clientSet == nil {
		return kernel.NewError(kernel.KindUnavailable, fmt.Errorf("cannot delete namespace %q: clientSet is nil", name))
	}

	if err := clientSet.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if kerrs.IsNotFound(err) {
			return nil
		}

		return kubeToDomainError(fmt.Errorf("failed to delete namespace %s: %w", name, err))

	}
	return nil
}

// List implements the persistence.ReaderRepo interface. params only needs to satisfy
// resource.ListFilter, so a resource with an extra scoping dimension (e.g. Network) can carry
// it on its own local params type and have it picked up here via the NetworkScope assertion,
// without that dimension living on the shared resource.ListParams struct.
func (a *ReaderAdapter[T]) List(ctx context.Context, params resource.ListFilter, list *[]T) (*string, error) {
	lo := metav1.ListOptions{}

	if limit := params.GetLimit(); limit > 0 {
		lo.Limit = int64(limit)
	}

	if skipToken := params.GetSkipToken(); skipToken != "" {
		lo.Continue = skipToken
	}

	selector := params.GetSelector()

	// Separate server-side and client-side selectors
	if selector != "" {
		lo.LabelSelector = filter.K8sSelectorForAPI(selector)
	}

	namespace, err := resolveNamespace(params)
	if err != nil {
		return nil, err
	}
	ri := a.client.Resource(a.gvr).Namespace(namespace)

	ulist, err := ri.List(ctx, lo)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to list resources", "resource", a.gvr.Resource, "error", err)
		return nil, kubeToDomainError(fmt.Errorf("failed to list resources for %s: %w", a.gvr.Resource, err))
	}

	// Apply client-side filtering for selectors not handled by the API
	var filteredItems []unstructured.Unstructured
	if selector != "" {
		for _, item := range ulist.Items {
			matched, k8sHandled, err := filter.MatchLabels(item.GetLabels(), selector)
			if err != nil {
				a.logger.ErrorContext(ctx, "label filter evaluation failed", "resource", a.gvr.Resource, "item", item.GetName(), "error", err)

				return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("label filter for %s failed: %w", a.gvr.Resource, err))
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

			return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert %s: %w", a.gvr.Resource, err))
		}

		*list = append(*list, converted)
	}

	next := ulist.GetContinue()
	if next == "" {
		return nil, nil
	}

	return &next, nil
}

// Load implements the persistence.ReaderRepo interface.
func (a *ReaderAdapter[T]) Load(ctx context.Context, obj *T) error {
	v := *obj
	namespace, err := resolveNamespace(v)
	if err != nil {
		return err
	}
	ri := a.client.Resource(a.gvr).Namespace(namespace)

	uobj, err := ri.Get(ctx, v.GetName(), metav1.GetOptions{})
	if err != nil {
		if !kerrs.IsNotFound(err) {
			a.logger.ErrorContext(ctx, "failed to get resource", "name", v.GetName(), "resource", a.gvr.Resource, "error", err)
		}
		return kubeToDomainError(fmt.Errorf("failed to retrieve %s '%s': %w", a.gvr.Resource, v.GetName(), err))
	}

	converted, err := a.k8sToDomain(uobj)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion failed", "resource", a.gvr.Resource, "error", err)
		return kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert %s: %w", a.gvr.Resource, err))
	}

	*obj = converted

	return nil
}

// Create implements the persistence.WriterRepo interface.
func (a *WriterAdapter[T]) Create(ctx context.Context, m T) (*T, error) {
	namespace, err := resolveNamespace(m)
	if err != nil {
		return nil, err
	}
	ri := a.client.Resource(a.gvr).Namespace(namespace)

	uobj, err := a.toUnstructured(m)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion to k8s object failed", "resource", a.gvr.Resource, "error", err)
		return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert %s to k8s object: %w", a.gvr.Resource, err))
	}

	ures, err := ri.Create(ctx, uobj, metav1.CreateOptions{})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create resource", "name", m.GetName(), "resource", a.gvr.Resource, "error", err)
		return nil, kubeToDomainError(fmt.Errorf("failed to create resource %s '%s': %w", a.gvr.Resource, m.GetName(), err))
	}

	res, err := a.k8sToDomain(ures)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion from k8s object failed", "resource", a.gvr.Resource, "error", err)
		return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert %s from k8s object: %w", a.gvr.Resource, err))
	}

	return &res, nil
}

// Update implements the persistence.WriterRepo interface. It updates the resource's
// metadata (labels, annotations) and spec. Status updates are handled separately
// by UpdateStatus.
func (a *WriterAdapter[T]) Update(ctx context.Context, m T) (*T, error) {
	uobj, err := a.toUnstructured(m)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion from T to unstructured failed", "resource", a.gvr.Resource, "error", err)
		return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert %s to unstructured: %w", a.gvr.Resource, err))
	}

	namespace, err := resolveNamespace(m)
	if err != nil {
		return nil, err
	}
	resourceInterface := a.client.Resource(a.gvr).Namespace(namespace)

	if m.GetVersion() == "" {
		if err := a.updateMetadataAndSpecRetry(ctx, resourceInterface, m.GetName(), uobj); err != nil {
			return nil, kubeToDomainError(fmt.Errorf("failed to update metadata and spec %s '%s': %w", a.gvr.Resource, m.GetName(), err))
		}
	} else {
		if _, err = resourceInterface.Update(ctx, uobj, metav1.UpdateOptions{}); err != nil {
			return nil, kubeToDomainError(fmt.Errorf("failed to update metadata and spec with version %s %s '%s': %w", m.GetVersion(), a.gvr.Resource, m.GetName(), err))
		}
	}

	currObj, err := resourceInterface.Get(ctx, m.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, kubeToDomainError(fmt.Errorf("failed to get %s '%s' after update: %w", a.gvr.Resource, m.GetName(), err))
	}

	res, err := a.k8sToDomain(currObj)
	if err != nil {
		return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert %s '%s' from k8s object: %w", a.gvr.Resource, m.GetName(), err))
	}

	return &res, nil
}

// UpdateStatus implements the persistence.WriterRepo interface. It updates only the
// resource's status subresource, leaving metadata and spec unchanged.
func (a *WriterAdapter[T]) UpdateStatus(ctx context.Context, m T) (*T, error) {
	uobj, err := a.toUnstructured(m)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion from T to unstructured failed", "resource", a.gvr.Resource, "error", err)
		return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert %s to unstructured: %w", a.gvr.Resource, err))
	}

	desiredStatus, statusFound, err := unstructured.NestedMap(uobj.Object, "status")
	if err != nil {
		return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to extract status from unstructured %s: %w", a.gvr.Resource, err))
	}

	if !statusFound {
		a.logger.ErrorContext(ctx, "no status field found in the provided object", "resource", a.gvr.Resource, "name", m.GetName())
		return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("no status data provided for %s '%s'", a.gvr.Resource, m.GetName()))
	}

	namespace, err := resolveNamespace(m)
	if err != nil {
		return nil, err
	}
	resourceInterface := a.client.Resource(a.gvr).Namespace(namespace)

	if err := a.updateStatusRetry(ctx, resourceInterface, m, desiredStatus); err != nil {
		a.logger.ErrorContext(ctx, "failed to update status", "resource", a.gvr.Resource, "error", err)
		return nil, kubeToDomainError(fmt.Errorf("failed to update status with retry %s '%s': %w", a.gvr.Resource, m.GetName(), err))
	}

	currObj, err := resourceInterface.Get(ctx, m.GetName(), metav1.GetOptions{})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get resource after status update", "resource", a.gvr.Resource, "error", err)
		return nil, kubeToDomainError(fmt.Errorf("failed to get %s '%s' after status update: %w", a.gvr.Resource, m.GetName(), err))
	}

	res, err := a.k8sToDomain(currObj)
	if err != nil {
		a.logger.ErrorContext(ctx, "conversion from k8s object failed", "resource", a.gvr.Resource, "error", err)
		return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert %s '%s' from k8s object after status update: %w", a.gvr.Resource, m.GetName(), err))
	}

	return &res, nil
}

func (a *WriterAdapter[T]) updateMetadataAndSpecRetry(
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

func (a *WriterAdapter[T]) updateStatusRetry(
	ctx context.Context,
	ri dynamic.ResourceInterface,
	m T,
	desiredStatus map[string]any,
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

// Delete implements the persistence.WriterRepo interface.
func (a *WriterAdapter[T]) Delete(ctx context.Context, m T) error {
	namespace, err := resolveNamespace(m)
	if err != nil {
		return err
	}
	ri := a.client.Resource(a.gvr).Namespace(namespace)

	deleteOptions := metav1.DeleteOptions{}
	if m.GetVersion() != "" {
		deleteOptions.Preconditions = &metav1.Preconditions{
			ResourceVersion: new(m.GetVersion()),
		}
	}

	if err := ri.Delete(ctx, m.GetName(), deleteOptions); err != nil {
		a.logger.ErrorContext(ctx, "failed to delete resource", "name", m.GetName(), "resource", a.gvr.Resource, "error", err, slog.Any("m", m))
		return kubeToDomainError(fmt.Errorf("failed to delete %s '%s': %w", a.gvr.Resource, m.GetName(), err))
	}

	return nil
}

// Delete refuses the delete when the child namespace still holds SECA resources,
// then deletes the resource CR and, if owned, the child namespace.
func (a *NamespaceManagingWriterAdapter[T]) Delete(ctx context.Context, m T) error {
	childNS, ownerLabels := childNamespaceFor(a.childNamespace, m)

	if childNS != "" {
		hasChildren, err := namespaceHasChildResources(ctx, a.client, childNS, a.childResourceGVRs)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to check child namespace emptiness",
				"namespace", childNS, "error", err)
			return err
		}
		if hasChildren {
			return kernel.NewError(kernel.KindConflict,
				fmt.Errorf("cannot delete %s %q: namespace %q is not empty", a.gvr.Resource, m.GetName(), childNS))
		}
	}

	if err := a.WriterAdapter.Delete(ctx, m); err != nil {
		return err
	}

	if childNS == "" {
		return nil
	}

	owned, err := namespaceOwnedBy(ctx, a.clientset, childNS, ownerLabels)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to verify namespace ownership after resource delete",
			"namespace", childNS, "error", err)
		return kubeToDomainError(fmt.Errorf("failed to verify ownership of namespace %q: %w", childNS, err))
	}
	if !owned {
		return nil
	}

	if err := DeleteNamespace(ctx, a.clientset, childNS); err != nil {
		a.logger.ErrorContext(ctx, "failed to delete owned child namespace",
			"namespace", childNS, "error", err)
		return err
	}
	return nil
}

// namespaceHasChildResources reports whether any of the given GVRs has at least one
// object in namespace. Missing CRDs (NotFound) are ignored so partial installs still work.
// An empty gvrs list means there is nothing to check — the namespace is treated as empty.
func namespaceHasChildResources(
	ctx context.Context,
	dyn dynamic.Interface,
	namespace string,
	gvrs []schema.GroupVersionResource,
) (bool, error) {
	if dyn == nil {
		return false, kernel.NewError(kernel.KindUnavailable, fmt.Errorf("cannot list child resources: dynamic client is nil"))
	}
	if namespace == "" || len(gvrs) == 0 {
		return false, nil
	}

	for _, gvr := range gvrs {
		list, err := dyn.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{Limit: 1})
		if err != nil {
			if kerrs.IsNotFound(err) {
				continue
			}
			return false, kubeToDomainError(fmt.Errorf("failed to list %s in namespace %q: %w", gvr.Resource, namespace, err))
		}
		if len(list.Items) > 0 {
			return true, nil
		}
	}
	return false, nil
}

// Watch implements the persistence.WatcherRepo interface.
func (a *WatcherAdapter[T]) Watch(ctx context.Context, m chan<- T) error {
	_ = ctx
	_ = m
	// TODO: implement the watch method of the kubernetes repo adapter.
	return kernel.NewError(kernel.KindUnavailable, errors.New("not implemented"))
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

// ChildNamespaceKind selects which namespace-provisioning behavior
// NamespaceManagingWriterAdapter.Create applies when creating a resource. It matches the two
// points in the tenant→workspace→network scope chain that own a namespace for their children:
// a Workspace provisions the namespace its children (Network, Nic, PublicIp, InternetGateway,
// ...) live in; a Network provisions the namespace its children (RouteTable, Subnet) live in.
// The current SECA resource organization
// (https://spec.secapi.cloud/docs/content/Architecture/resource-organization) has no other
// point on the chain that owns a children namespace, so this is a closed set rather than an
// injected function.
type ChildNamespaceKind int

const (
	// NoChildNamespace opts out: Create delegates straight to the wrapped WriterAdapter.
	NoChildNamespace ChildNamespaceKind = iota
	// WorkspaceChildren provisions the tenant/workspace namespace, using the resource's own
	// name as the workspace segment (there is no Tenant entity, so the Workspace is the only
	// entity that manages this).
	WorkspaceChildren
	// NetworkChildren provisions the tenant/workspace/network namespace, using the resource's
	// own name as the network segment.
	NetworkChildren
)

// namespaceScope is a plain NetworkScope value used to call ComputeNetworkNamespace from raw
// tenant/workspace/network strings, without needing a full domain object.
type namespaceScope struct {
	tenant, workspace, network string
}

func (s namespaceScope) GetTenant() string    { return s.tenant }
func (s namespaceScope) GetWorkspace() string { return s.workspace }
func (s namespaceScope) GetNetwork() string   { return s.network }

// childNamespaceFor computes the namespace (and owner labels) a resource's children will live
// in, for the given ChildNamespaceKind. Returns "" for NoChildNamespace.
func childNamespaceFor(kind ChildNamespaceKind, m persistence.IdentifiableResource) (namespace string, ownerLabels map[string]string) {
	switch kind {
	case WorkspaceChildren:
		return childNamespaceLabels(m.GetTenant(), m.GetName(), "")
	case NetworkChildren:
		return childNamespaceLabels(m.GetTenant(), m.GetWorkspace(), m.GetName())
	case NoChildNamespace:
		return "", nil
	default:
		return "", nil
	}
}

// childNamespaceLabels builds the namespace and its owner labels for a tenant/workspace[/network]
// triple, hashing via ComputeNetworkNamespace when network is set and ComputeNamespace otherwise.
func childNamespaceLabels(tenant, workspace, network string) (string, map[string]string) {
	ownerLabels := map[string]string{}
	if tenant != "" {
		ownerLabels[labels.InternalTenantLabel] = tenant
	}
	if workspace != "" {
		ownerLabels[labels.InternalWorkspaceLabel] = workspace
	}

	if network != "" {
		ownerLabels[labels.InternalNetworkLabel] = network
		return ComputeNetworkNamespace(namespaceScope{tenant: tenant, workspace: workspace, network: network}), ownerLabels
	}
	return ComputeNamespace(&resource.Scope{Tenant: tenant, Workspace: workspace}), ownerLabels
}

// NamespaceManagingWriterAdapter wraps a WriterAdapter and ensures the namespace computed for
// childNamespace exists before creating resources, rolling back that namespace if it created it
// and the resource creation subsequently fails. On Delete it refuses when childResourceGVRs
// still list objects in that namespace, then deletes the CR and the owned child namespace.
// It uses a typed clientset for Namespace operations when available.
type NamespaceManagingWriterAdapter[T persistence.IdentifiableResource] struct {
	*WriterAdapter[T]
	client            dynamic.Interface
	clientset         kubernetes.Interface
	logger            *slog.Logger
	childNamespace    ChildNamespaceKind
	childResourceGVRs []schema.GroupVersionResource
}

// NamespaceManagingRepoAdapter implements the persistence.WatcherRepo interface for a specific resource type.
type NamespaceManagingRepoAdapter[T persistence.IdentifiableResource] struct {
	*ReaderAdapter[T]
	*NamespaceManagingWriterAdapter[T]
	*WatcherAdapter[T]
}

// NewNamespaceManagingWriterAdapter creates a new writer adapter that ensures the namespace
// selected by childNamespace exists before creating resources.
// childResourceGVRs is the closed set of SECA types that may live in the child namespace;
// Delete uses it for the emptiness check (empty/nil means no types to check).
func NewNamespaceManagingWriterAdapter[T persistence.IdentifiableResource](
	dynClient dynamic.Interface,
	clientset kubernetes.Interface,
	gvr schema.GroupVersionResource,
	logger *slog.Logger,
	domainToK8s DomainToK8s[T],
	k8sToDomain K8sToDomain[T],
	childNamespace ChildNamespaceKind,
	childResourceGVRs []schema.GroupVersionResource,
) *NamespaceManagingWriterAdapter[T] {
	base := NewWriterAdapter(dynClient, gvr, logger, domainToK8s, k8sToDomain)
	return &NamespaceManagingWriterAdapter[T]{
		WriterAdapter:     base,
		client:            dynClient,
		clientset:         clientset,
		logger:            logger,
		childNamespace:    childNamespace,
		childResourceGVRs: childResourceGVRs,
	}
}

// NewNamespaceManagingRepoAdapter creates a new Kubernetes adapter for the persistence.WriterRepo port.
func NewNamespaceManagingRepoAdapter[T persistence.IdentifiableResource](
	dynClient dynamic.Interface,
	clientset kubernetes.Interface,
	gvr schema.GroupVersionResource,
	logger *slog.Logger,
	domainToK8s DomainToK8s[T],
	k8sToDomain K8sToDomain[T],
	childNamespace ChildNamespaceKind,
	childResourceGVRs []schema.GroupVersionResource,
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
			childNamespace,
			childResourceGVRs,
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

// Create ensures the namespace selected by a.childNamespace exists and rolls back if it
// created that namespace and the resource creation subsequently fails.
func (a *NamespaceManagingWriterAdapter[T]) Create(ctx context.Context, m T) (*T, error) {
	namespace, ownerLabels := childNamespaceFor(a.childNamespace, m)
	if namespace == "" {
		return a.WriterAdapter.Create(ctx, m)
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
