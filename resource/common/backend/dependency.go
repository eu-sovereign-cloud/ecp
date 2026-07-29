package backend

import (
	"context"
	"fmt"
	"strings"

	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// ReferenceTarget identifies a single resource resolved from a domain.Reference:
// the namespace scope (tenant/workspace) and the resource name.
type ReferenceTarget struct {
	Tenant    string
	Workspace string
	Name      string
}

// ParseReference resolves a domain.Reference into its tenant, workspace, and name.
// Tenant and workspace are read from the reference, whether carried as explicit
// fields or embedded in the resource path; an empty tenant falls back to defaultTenant.
// The name is the last segment of the resource path (e.g. "block-storages/web" -> "web").
func ParseReference(ref domain.Reference, defaultTenant string) ReferenceTarget {
	cr := ReferenceToCR(ref)

	tenant := cr.Tenant
	if tenant == "" {
		tenant = defaultTenant
	}

	name := cr.Resource
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}

	return ReferenceTarget{Tenant: tenant, Workspace: cr.Workspace, Name: name}
}

// ReferenceResolver resolves cross-resource dependencies against the Kubernetes API
// using a dynamic client. It reads only the well-known status.state field and the
// structured reference fields of target resources, so it needs no per-resource
// converters and introduces no import cycle between resource slices.
type ReferenceResolver struct {
	client dynamic.Interface
}

// NewReferenceResolver creates a ReferenceResolver backed by the given dynamic client.
func NewReferenceResolver(client dynamic.Interface) *ReferenceResolver {
	return &ReferenceResolver{client: client}
}

// State resolves ref to a single resource of the given GVR and reports whether it
// exists together with its lifecycle state. defaultTenant is used when the reference
// does not carry an explicit tenant. A not-found resource returns (false, "", nil).
func (rr *ReferenceResolver) State(
	ctx context.Context,
	gvr schema.GroupVersionResource,
	ref domain.Reference,
	defaultTenant string,
) (bool, domain.ResourceState, error) {
	target := ParseReference(ref, defaultTenant)
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: target.Tenant, Workspace: target.Workspace})

	obj, err := rr.client.Resource(gvr).Namespace(namespace).Get(ctx, target.Name, metav1.GetOptions{})
	if err != nil {
		if kerrs.IsNotFound(err) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("%s %s: failed to resolve reference: %w", gvr.Resource, target.Name, err)
	}

	raw, _, err := unstructured.NestedString(obj.Object, "status", "state")
	if err != nil {
		return false, "", fmt.Errorf("%s %s: failed to read status state: %w", gvr.Resource, target.Name, err)
	}

	return true, ResourceStateFromCR(schemav1.ResourceState(raw)), nil
}

// Referrers lists resources of the given GVR in namespace and returns the names of
// those whose reference at fieldPath (e.g. {"spec","blockStorageRef"}) points at the
// target. defaultTenant is used when a listed reference omits its tenant.
func (rr *ReferenceResolver) Referrers(
	ctx context.Context,
	gvr schema.GroupVersionResource,
	namespace string,
	fieldPath []string,
	target ReferenceTarget,
	defaultTenant string,
) ([]string, error) {
	list, err := rr.client.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to list referrers: %w", gvr.Resource, err)
	}

	var names []string
	for i := range list.Items {
		item := &list.Items[i]

		refMap, found, err := unstructured.NestedMap(item.Object, fieldPath...)
		if err != nil {
			return nil, fmt.Errorf("%s %s: failed to read reference: %w", gvr.Resource, item.GetName(), err)
		}
		if !found {
			continue
		}

		var crRef schemav1.Reference
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(refMap, &crRef); err != nil {
			return nil, fmt.Errorf("%s %s: failed to convert reference: %w", gvr.Resource, item.GetName(), err)
		}

		got := ParseReference(ReferenceFromCR(crRef), defaultTenant)
		if got.Tenant == target.Tenant && got.Workspace == target.Workspace && got.Name == target.Name {
			names = append(names, item.GetName())
		}
	}

	return names, nil
}
