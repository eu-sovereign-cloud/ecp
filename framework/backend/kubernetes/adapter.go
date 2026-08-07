package kubernetes

import (
	"context"
	"crypto/sha3"
	"encoding/json"
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
	"k8s.io/apimachinery/pkg/types"
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
		if !kerrs.IsAlreadyExists(err) {
			return false, kubeToDomainError(fmt.Errorf("failed to create namespace %s: %w", name, err))
		}

		// The namespace exists but may predate the owner labels (a hand-applied dev fixture, a
		// leftover from an older release). Without them the owning controller cannot prove the
		// namespace is its own and refuses to delete it, so it would leak with no path back.
		// The name is a hash of exactly this scope, so stamping is safe; the merge patch leaves
		// any other label alone.
		if len(ownerLabels) > 0 {
			if err := patchNamespaceLabels(ctx, clientSet, name, ownerLabels); err != nil {
				return false, err
			}
		}

		return false, nil
	}

	return true, nil
}

// patchNamespaceLabels merges ownerLabels into the namespace's labels, leaving the rest untouched.
func patchNamespaceLabels(ctx context.Context, clientSet kubernetes.Interface, name string, ownerLabels map[string]string) error {
	patch, err := json.Marshal(map[string]any{"metadata": map[string]any{"labels": ownerLabels}})
	if err != nil {
		return kernel.NewError(kernel.KindInternal, fmt.Errorf("failed to build label patch for namespace %s: %w", name, err))
	}

	if _, err := clientSet.CoreV1().Namespaces().Patch(
		ctx, name, types.MergePatchType, patch, metav1.PatchOptions{},
	); err != nil && !kerrs.IsNotFound(err) {
		return kubeToDomainError(fmt.Errorf("failed to stamp owner labels on namespace %s: %w", name, err))
	}

	return nil
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
		// A versioned update is a full replace, and uobj carries only what the domain type
		// models. Finalizers are not part of it, so replacing blind deletes them — and a
		// resource whose deletion is already under way has nothing left holding it, so the API
		// server reclaims it on the spot, before its controller's cleanup hook has run. That is
		// silent: the plugin is mid-delete and simply never gets another reconcile.
		if err := a.preserveFinalizers(ctx, resourceInterface, m.GetName(), uobj); err != nil {
			return nil, err
		}

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

// preserveFinalizers copies the stored object's finalizers onto desired, so a full-replace
// update cannot drop them. Finalizers belong to whoever registered them — the controller, not
// the caller of Update — and no domain type models them, so they can only be carried across
// from the live object. A NotFound is left to the Update that follows, which reports it in the
// caller's own terms.
func (a *WriterAdapter[T]) preserveFinalizers(
	ctx context.Context,
	ri dynamic.ResourceInterface,
	name string,
	desired *unstructured.Unstructured,
) error {
	curr, err := ri.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if kerrs.IsNotFound(err) {
			return nil
		}

		return kubeToDomainError(fmt.Errorf("failed to read %s '%s' before update: %w", a.gvr.Resource, name, err))
	}

	if fins := curr.GetFinalizers(); len(fins) > 0 {
		desired.SetFinalizers(fins)
	}

	return nil
}

func (a *WriterAdapter[T]) updateMetadataAndSpecRetry(
	ctx context.Context,
	ri dynamic.ResourceInterface,
	name string,
	desired *unstructured.Unstructured,
) error {
	desiredLabels := desired.GetLabels()
	desiredAnnotations := desired.GetAnnotations()

	// Both subtrees are extracted once, up front, not per attempt. NestedMap deep-copies what it
	// returns, so pulling them inside the closure would re-copy them on every conflict retry, and a
	// structurally invalid desired object would only be caught after a Get round trip rather than
	// failing immediately.
	//
	// commonData is a sibling of spec, not part of it, so it needs copying in its own right. It is
	// not cosmetic: commonData.labels holds the *key list* that KeyedToOriginal walks to rebuild a
	// resource's labels from the hashed kl/<sha3> entries in metadata.labels. Leaving it behind
	// means a newly added label key never appears in the domain object - the value is written to
	// metadata.labels but nothing knows to look it up again.
	desiredSpec, specFound, err := unstructured.NestedMap(desired.Object, "spec")
	if err != nil {
		return err
	}

	desiredCommonData, commonDataFound, err := unstructured.NestedMap(desired.Object, "commonData")
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

		specChanged, err := syncNestedMap(currObj, desiredSpec, specFound, "spec")
		if err != nil {
			return err
		}

		commonDataChanged, err := syncNestedMap(currObj, desiredCommonData, commonDataFound, "commonData")
		if err != nil {
			return err
		}

		labelsChanged := !cmp.Equal(currObj.GetLabels(), desiredLabels)
		if labelsChanged {
			currObj.SetLabels(desiredLabels)
		}

		annotationsChanged := !cmp.Equal(currObj.GetAnnotations(), desiredAnnotations)
		if annotationsChanged {
			currObj.SetAnnotations(desiredAnnotations)
		}

		if !specChanged && !commonDataChanged && !labelsChanged && !annotationsChanged {
			return nil
		}

		_, err = ri.Update(ctx, currObj, metav1.UpdateOptions{})

		return err
	})
}

// syncNestedMap copies an already-extracted desired value onto curr's named top-level field when
// the two differ, reporting whether it wrote. spec and its sibling commonData get identical
// treatment, so they share one path. A field absent from desired (found=false) is left alone rather
// than cleared.
//
// The comparison is only as stable as what the converters produce: an equal-but-differently-ordered
// value counts as a change here and costs a write, a resourceVersion bump, and the reconcile that
// follows it. commonData.labels is a list built from a Go map, so the converters sort it - see
// doc/CONVENTIONS.md.
func syncNestedMap(curr *unstructured.Unstructured, desiredValue map[string]any, found bool, name string) (bool, error) {
	if !found {
		return false, nil
	}

	currValue, currFound, err := unstructured.NestedMap(curr.Object, name)
	if err != nil {
		return false, err
	}

	if currFound && cmp.Equal(currValue, desiredValue) {
		return false, nil
	}

	return true, unstructured.SetNestedMap(curr.Object, desiredValue, name)
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

// Delete refuses the delete when the child namespace still holds SECA resources, then deletes the
// resource CR.
//
// Only the refusal lives here. It is a user-facing invariant ("a workspace with resources in it
// cannot be deleted") and has to answer synchronously with 409, so it cannot become eventually
// consistent. Tearing the namespace down afterwards is a side effect with no caller waiting on it
// and belongs to the owning controller's finalizer — see NamespaceCleanup.
func (a *NamespaceManagingWriterAdapter[T]) Delete(ctx context.Context, m T) error {
	childNS, _ := childNamespaceFor(a.childNamespace, m)

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

	return a.WriterAdapter.Delete(ctx, m)
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
	// NoChildNamespace opts out of the child namespace: Create still ensures the tenant
	// namespace the resource itself lives in (the global gateway's Role/RoleAssignment case),
	// so it is not the same as a plain WriterAdapter.
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
		return namespaceOwnerLabels(m.GetTenant(), m.GetName(), "")
	case NetworkChildren:
		return namespaceOwnerLabels(m.GetTenant(), m.GetWorkspace(), m.GetName())
	case NoChildNamespace:
		return "", nil
	default:
		return "", nil
	}
}

// namespaceOwnerLabels builds the namespace and its owner labels for a tenant/workspace[/network]
// triple, hashing via ComputeNetworkNamespace when network is set and ComputeNamespace otherwise.
func namespaceOwnerLabels(tenant, workspace, network string) (string, map[string]string) {
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

// NamespaceCleanup returns a controller cleanup hook (see controller.GenericController.WithCleanup)
// that tears down the namespace a resource owns for its children.
//
// It lives on the controller rather than the API write path so the finalizer can retry it, and so
// it runs after the plugin has finished deleting instead of racing ahead of it. The emptiness
// check the write path already did is repeated because that one ran in another process and a
// namespace delete is irreversible and cascades.
func NamespaceCleanup[T persistence.IdentifiableResource](
	dyn dynamic.Interface,
	clientset kubernetes.Interface,
	logger *slog.Logger,
	childNamespace ChildNamespaceKind,
	childResourceGVRs []schema.GroupVersionResource,
) func(context.Context, T) error {
	return func(ctx context.Context, m T) error {
		namespace, ownerLabels := childNamespaceFor(childNamespace, m)
		if namespace == "" {
			return nil
		}

		hasChildren, err := namespaceHasChildResources(ctx, dyn, namespace, childResourceGVRs)
		if err != nil {
			return err
		}
		if hasChildren {
			// Anomalous: the write path should have refused this delete. Erroring keeps the
			// finalizer on, so the resource stays in Terminating until the children are gone
			// rather than taking them down with it.
			return kernel.NewError(kernel.KindConflict,
				fmt.Errorf("cannot delete namespace %q of %s: it still holds resources", namespace, m.GetName()))
		}

		owned, err := namespaceOwnedBy(ctx, clientset, namespace, ownerLabels)
		if kerrs.IsNotFound(err) {
			// Already gone — a retry after a successful delete, or a namespace that was never
			// provisioned. Nothing to do, and nothing to warn about.
			return nil
		}
		if err != nil {
			return kubeToDomainError(fmt.Errorf("failed to verify ownership of namespace %q: %w", namespace, err))
		}
		if !owned {
			// A namespace we did not label is not ours to delete. Not an error: retrying would
			// spin on a finalizer that can never be satisfied. It leaks, so say so out loud.
			logger.WarnContext(ctx, "leaving namespace in place: owner labels do not match",
				"namespace", namespace, "resource", m.GetName(), "expected_labels", ownerLabels)
			return nil
		}

		return DeleteNamespace(ctx, clientset, namespace)
	}
}

// NamespaceManagingWriterAdapter wraps a WriterAdapter and, on Create, ensures the tenant
// namespace and the one computed for childNamespace exist. On Delete it refuses when
// childResourceGVRs still list objects in that namespace, then deletes the CR — the namespace
// itself is torn down by the owning controller's NamespaceCleanup finalizer.
// It uses a typed clientset for Namespace operations when available.
// The dynamic client and logger are read through the embedded WriterAdapter's Adapter — keeping
// a second copy here would be two fields for one value, free to drift apart.
type NamespaceManagingWriterAdapter[T persistence.IdentifiableResource] struct {
	*WriterAdapter[T]
	clientset         kubernetes.Interface
	childNamespace    ChildNamespaceKind
	childResourceGVRs []schema.GroupVersionResource
}

// NamespaceManagingRepoAdapter implements the persistence.WatcherRepo interface for a specific resource type.
type NamespaceManagingRepoAdapter[T persistence.IdentifiableResource] struct {
	*ReaderAdapter[T]
	*NamespaceManagingWriterAdapter[T]
	*WatcherAdapter[T]
}

// NewNamespaceManagingWriterAdapter creates a new writer adapter that ensures the tenant
// namespace and the namespace selected by childNamespace exist before creating resources.
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
		clientset:         clientset,
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
//
// A missing namespace is returned as a NotFound error rather than "not owned": the two mean
// opposite things to a caller deciding whether to delete, and conflating them makes an
// already-deleted namespace look like someone else's.
func namespaceOwnedBy(ctx context.Context, clientset kubernetes.Interface, nsName string, expectedLabels map[string]string) (bool, error) {
	if clientset == nil {
		return false, fmt.Errorf("clientset is nil")
	}

	ns, err := clientset.CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
	if err != nil {
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

// Create ensures the namespaces the resource needs exist, then creates it.
//
// The resource's own namespace is provisioned only when it is the tenant namespace. There is no
// Tenant entity, so nobody else would ever create it. Below that level a namespace is owned by a
// parent entity and its absence *is* the referential-integrity check: fabricating it would let a
// Network land in a Workspace that was never created, in a namespace no controller would ever
// reclaim. A resource whose scope names a workspace therefore fails with NotFound, exactly as a
// leaf resource on a plain WriterAdapter does.
//
// The child namespace is rolled back if this call created it and the CR create then fails —
// without an owning CR nothing will ever reclaim it, and the caller picks its name.
func (a *NamespaceManagingWriterAdapter[T]) Create(ctx context.Context, m T) (*T, error) {
	if m.GetWorkspace() == "" {
		tenantNS, tenantLabels := namespaceOwnerLabels(m.GetTenant(), "", "")
		if tenantNS != "" {
			if _, err := CreateNamespace(ctx, a.clientset, tenantNS, tenantLabels); err != nil {
				return nil, err
			}
		}
	}

	childNS, childLabels := childNamespaceFor(a.childNamespace, m)
	if childNS == "" {
		return a.WriterAdapter.Create(ctx, m)
	}

	createdNS, err := CreateNamespace(ctx, a.clientset, childNS, childLabels)
	if err != nil {
		return nil, err
	}

	res, err := a.WriterAdapter.Create(ctx, m)
	if err != nil && createdNS {
		if owned, ownErr := namespaceOwnedBy(ctx, a.clientset, childNS, childLabels); ownErr == nil && owned {
			if delErr := DeleteNamespace(ctx, a.clientset, childNS); delErr != nil {
				a.logger.ErrorContext(ctx, "failed to roll back namespace created for resource",
					"namespace", childNS, "error", delErr)
			}
		}
	}

	return res, err
}
