//go:build envtest

package kubernetes_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

	. "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role/backend/kubernetes"
)

func TestRoleBackend_CreateAndGetRole(t *testing.T) {
	t.Parallel()

	// Use a config copy with higher rate limits to avoid rate limiter exhaustion.
	testCfg := rest.CopyConfig(cfg)
	testCfg.QPS = 50
	testCfg.Burst = 100
	dynClient, err := dynamic.NewForConfig(testCfg)
	require.NoError(t, err)

	// Keep tenant short to fit within 63 char label limit.
	tenant := "t-role-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	namespace := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: tenant})
	const roleName = "test-role"
	namespaceGVR := k8sschema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

	// Create the namespace so the CRD resources have somewhere to land.
	namespaceObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": namespace,
			},
		},
	}

	ctx := context.Background()
	_, err = dynClient.Resource(namespaceGVR).Create(ctx, namespaceObj, metav1.CreateOptions{})
	require.NoError(t, err)

	// Cleanup namespace and all resources within it.
	t.Cleanup(func() {
		_ = dynClient.Resource(namespaceGVR).Delete(context.Background(), namespace, metav1.DeleteOptions{})
	})

	// Build writer and reader adapters directly from the kubernetes backend package.
	writerRepo := k8sadapter.NewWriterAdapter[*roledom.Role](
		dynClient,
		RoleGVR,
		slog.Default(),
		RoleToCR,
		RoleFromCR,
	)
	readerRepo := k8sadapter.NewReaderAdapter[*roledom.Role](
		dynClient,
		RoleGVR,
		slog.Default(),
		RoleFromCR,
	)

	// Create a role with one permission.
	role := &roledom.Role{
		GlobalTenantMetadata: commondomain.GlobalTenantMetadata{
			CommonMetadata: commondomain.CommonMetadata{
				Name:     roleName,
				Provider: roledom.ProviderID,
			},
			Scope: kernelresource.Scope{Tenant: tenant},
		},
		Spec: roledom.RoleSpec{
			Permissions: []roledom.Permission{
				{
					Provider:  "seca.compute",
					Resources: []string{"instances"},
					Verb:      []string{"get", "list"},
				},
			},
		},
	}

	created, err := writerRepo.Create(ctx, role)
	require.NoError(t, err)
	require.Equal(t, roleName, created.Name)
	require.Equal(t, tenant, created.Tenant)
	require.Equal(t, 1, len(created.Spec.Permissions))
	require.Equal(t, "seca.compute", created.Spec.Permissions[0].Provider)

	// Fetch the role back.
	got, err := readerRepo.Get(ctx, created)
	require.NoError(t, err)
	require.Equal(t, roleName, got.Name)
	require.Equal(t, tenant, got.Tenant)
	require.Equal(t, 1, len(got.Spec.Permissions))
	require.Equal(t, "seca.compute", got.Spec.Permissions[0].Provider)
}
