package testenv

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

// WorkspaceRegion reads the Workspace CR tenant/name keeps in region's namespace and returns the
// region it carries as its field and as its internal region label. The two must always be equal:
// the field is what a plugin provisions into, the label what the gateway's list filter and a
// delegator's region scope select on.
func WorkspaceRegion(ctx context.Context, dyn dynamic.Interface, tenant, name, region string) (field, label string, err error) {
	namespace := k8sadapter.ComputeRegionNamespace(&wsdom.Workspace{
		RegionalMetadata: commondomain.RegionalMetadata{
			Scope:  kernelresource.Scope{Tenant: tenant},
			Region: region,
		},
	})

	cr, err := dyn.Resource(wsk8s.WorkspaceGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	field, _, err = unstructured.NestedString(cr.Object, "region")

	return field, cr.GetLabels()[k8slabels.InternalRegionLabel], err
}
