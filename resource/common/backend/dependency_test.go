package backend_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// The fake dynamic client guesses a resource name from the kind (BlockStorage ->
// blockstorages); these synthetic GVRs use the guessed plural forms so Get/List resolve.
var (
	bsGVR  = schema.GroupVersionResource{Group: "storage.test", Version: "v1", Resource: "blockstorages"}
	imgGVR = schema.GroupVersionResource{Group: "storage.test", Version: "v1", Resource: "images"}
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		name          string
		ref           commondomain.Reference
		defaultTenant string
		want          commonbackend.ReferenceTarget
	}{
		{
			name:          "explicit workspace field, tenant inferred",
			ref:           commondomain.Reference{Workspace: "w1", Resource: "block-storages/bs1"},
			defaultTenant: "t1",
			want:          commonbackend.ReferenceTarget{Tenant: "t1", Workspace: "w1", Name: "bs1"},
		},
		{
			name:          "tenant and workspace embedded in path",
			ref:           commondomain.Reference{Resource: "tenants/t2/workspaces/w2/block-storages/bs2"},
			defaultTenant: "t1",
			want:          commonbackend.ReferenceTarget{Tenant: "t2", Workspace: "w2", Name: "bs2"},
		},
		{
			name:          "tenant-only reference",
			ref:           commondomain.Reference{Resource: "images/img1"},
			defaultTenant: "t1",
			want:          commonbackend.ReferenceTarget{Tenant: "t1", Workspace: "", Name: "img1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, commonbackend.ParseReference(tc.ref, tc.defaultTenant))
		})
	}
}

func TestReferenceResolver_State(t *testing.T) {
	tenant, workspace := "t1", "w1"
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant, Workspace: workspace})

	bsActive := newRefObject("storage.test/v1", "BlockStorage", namespace, "bs1", "active", nil)

	dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds(), bsActive)
	resolver := commonbackend.NewReferenceResolver(dynFake)

	t.Run("returns state of an existing reference", func(t *testing.T) {
		ref := commondomain.Reference{Workspace: workspace, Resource: "block-storages/bs1"}
		exists, state, err := resolver.State(context.Background(), bsGVR, ref, tenant)
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, commondomain.ResourceStateActive, state)
	})

	t.Run("reports a missing reference as not found", func(t *testing.T) {
		ref := commondomain.Reference{Workspace: workspace, Resource: "block-storages/missing"}
		exists, state, err := resolver.State(context.Background(), bsGVR, ref, tenant)
		require.NoError(t, err)
		require.False(t, exists)
		require.Empty(t, state)
	})
}

func TestReferenceResolver_Referrers(t *testing.T) {
	tenant, workspace := "t1", "w1"
	imageNamespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant})

	bsRef := map[string]any{"workspace": workspace, "resource": "block-storages/bs1"}
	otherRef := map[string]any{"workspace": workspace, "resource": "block-storages/other"}

	referring := newRefObject("storage.test/v1", "Image", imageNamespace, "img-referring", "active", bsRef)
	unrelated := newRefObject("storage.test/v1", "Image", imageNamespace, "img-unrelated", "active", otherRef)

	dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds(), referring, unrelated)
	resolver := commonbackend.NewReferenceResolver(dynFake)

	target := commonbackend.ReferenceTarget{Tenant: tenant, Workspace: workspace, Name: "bs1"}
	names, err := resolver.Referrers(context.Background(), imgGVR, imageNamespace, []string{"spec", "blockStorageRef"}, target, tenant)
	require.NoError(t, err)
	require.Equal(t, []string{"img-referring"}, names)
}

func listKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{
		bsGVR:  "BlockStorageList",
		imgGVR: "ImageList",
	}
}

// newRefObject builds an unstructured CR carrying a status.state and, optionally, a
// spec.<ref> structured reference, for driving the ReferenceResolver against a fake client.
func newRefObject(apiVersion, kind, namespace, name, state string, ref map[string]any) *unstructured.Unstructured {
	spec := map[string]any{}
	if ref != nil {
		spec["blockStorageRef"] = ref
	}

	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   map[string]any{"namespace": namespace, "name": name},
		"spec":       spec,
		"status":     map[string]any{"state": state},
	}}
}
