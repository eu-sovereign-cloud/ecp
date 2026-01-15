package workspace

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes/labels"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

var cfg *rest.Config

// --- Envtest lifecycle ---
func TestMain(m *testing.M) {
	wd, _ := os.Getwd()
	crdDir := filepath.Clean(filepath.Join(wd, "../../../../../api/generated/crds/workspace"))
	testEnvironment := &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     []string{crdDir},
		DownloadBinaryAssets:  true,
		BinaryAssetsDirectory: filepath.Join(os.TempDir(), "envtest-binaries"),
	}
	var err error
	cfg, err = testEnvironment.Start()
	if err != nil {
		slog.Error("failed to start envtest", "error", err)
		os.Exit(1)
	}
	code := m.Run()
	if err := testEnvironment.Stop(); err != nil {
		slog.Error("failed to stop envtest", "error", err)
	}
	os.Exit(code)
}

func TestWorkspaceController(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, storage.AddToScheme(scheme))

	dynClient, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)

	// Create valid Kubernetes namespace name (lowercase, alphanumeric and hyphens only)
	// Keep tenant short to fit within 63 char label limit
	tenant := "t-workspace-" + strings.ToLower(strings.ReplaceAll(t.Name(), "_", "-"))
	if len(tenant) > 63 {
		tenant = tenant[:63]
	}
	namespace := kubernetes.ComputeNamespace(&scope.Scope{Tenant: tenant})
	const workspaceName = "test-workspace"
	namespaceGVR := k8sschema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

	// Create the namespace object
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

	// Cleanup namespace and all resources within it
	t.Cleanup(func() {
		_ = dynClient.Resource(namespaceGVR).Delete(context.Background(), namespace, metav1.DeleteOptions{})
	})

	writerRepo := kubernetes.NewWriterAdapter(
		dynClient,
		workspacev1.WorkspaceGVR,
		slog.Default(),
		kubernetes.MapWorkspaceDomainToCR,
		kubernetes.MapCRToWorkspaceDomain,
	)

	readerRepo := kubernetes.NewReaderAdapter(
		dynClient,
		workspacev1.WorkspaceGVR,
		slog.Default(),
		kubernetes.MapCRToWorkspaceDomain,
	)

	// Setup controllers
	createController := CreateWorkspace{
		Logger: slog.Default(),
		Repo:   writerRepo,
	}

	getController := GetWorkspace{
		Logger: slog.Default(),
		Repo:   readerRepo,
	}

	deleteController := DeleteWorkspace{
		Logger: slog.Default(),
		Repo:   writerRepo,
	}

	listController := ListWorkspace{
		Logger: slog.Default(),
		Repo:   readerRepo,
	}

	updateController := UpdateWorkspace{
		Logger: slog.Default(),
		Repo:   writerRepo,
	}

	t.Run("create_workspace", func(t *testing.T) {
		// Create a workspace domain object
		createDomain := &regional.WorkspaceDomain{
			Metadata: regional.Metadata{
				CommonMetadata: model.CommonMetadata{
					Name: workspaceName,
				},
				Scope: scope.Scope{
					Tenant: tenant,
				},
				Labels: map[string]string{
					labels.InternalTenantLabel: tenant,
				},
			},
			Spec: regional.WorkspaceSpec{
				"test-string": "test-value",
				"test-number": int64(42),
				"test-bool":   true,
				"test-list":   []string{"a", "b", "c"},
				"test-map": map[string]interface{}{
					"inner-string": "inner-value",
					"inner-number": int64(7),
					"inner-bool":   false,
					"inner-list":   []int64{1, 2, 3},
				},
			},
		}

		// Create the workspace
		createdDomain, err := createController.Do(ctx, createDomain)
		require.NoError(t, err)
		require.NotNil(t, createdDomain)
		require.Equal(t, workspaceName, createdDomain.Name)
		require.Equal(t, "test-value", createdDomain.Spec["test-string"])
		require.Equal(t, int64(42), createdDomain.Spec["test-number"])
		require.Equal(t, true, createdDomain.Spec["test-bool"])
		require.Equal(t, []interface{}{"a", "b", "c"}, createdDomain.Spec["test-list"])
		require.Equal(t, map[string]interface{}{
			"inner-string": "inner-value",
			"inner-number": int64(7),
			"inner-bool":   false,
			"inner-list":   []interface{}{int64(1), int64(2), int64(3)},
		}, createdDomain.Spec["test-map"])
	})

	t.Run("get_workspace", func(t *testing.T) {
		// Get the workspace and verify it matches
		metadata := regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: workspaceName,
			},
			Scope: scope.Scope{
				Tenant: tenant,
			},
		}
		retrievedDomain, err := getController.Do(ctx, &metadata)
		require.NoError(t, err)
		require.NotNil(t, retrievedDomain)
		require.Equal(t, workspaceName, retrievedDomain.Name)
		require.Equal(t, "test-value", retrievedDomain.Spec["test-string"])
		require.Equal(t, int64(42), retrievedDomain.Spec["test-number"])
		require.Equal(t, true, retrievedDomain.Spec["test-bool"])
		require.Equal(t, []interface{}{"a", "b", "c"}, retrievedDomain.Spec["test-list"])
		require.Equal(t, map[string]interface{}{
			"inner-string": "inner-value",
			"inner-number": int64(7),
			"inner-bool":   false,
			"inner-list":   []interface{}{int64(1), int64(2), int64(3)},
		}, retrievedDomain.Spec["test-map"])
	})

	t.Run("get_nonexistent_workspace", func(t *testing.T) {
		metadata := regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: "missing-workspace",
			},
			Scope: scope.Scope{
				Tenant: tenant,
			},
		}
		_, err := getController.Do(ctx, &metadata)
		require.Error(t, err)
	})

	t.Run("list_workspace", func(t *testing.T) {
		workspaces, _, err := listController.Do(ctx, model.ListParams{Scope: scope.Scope{Tenant: tenant}})
		require.NoError(t, err)
		require.Len(t, workspaces, 1)
		require.Equal(t, workspaceName, workspaces[0].Name)
	})

	t.Run("update_workspace", func(t *testing.T) {
		// Update the workspace
		updateDomain := &regional.WorkspaceDomain{
			Metadata: regional.Metadata{
				CommonMetadata: model.CommonMetadata{
					Name: workspaceName,
				},
				Scope: scope.Scope{
					Tenant: tenant,
				},
			},
			Spec: regional.WorkspaceSpec{
				"test-string": "updated-value",
				"test-number": int64(84),
			},
		}

		updatedDomain, err := updateController.Do(ctx, updateDomain)
		require.NoError(t, err)
		require.NotNil(t, updatedDomain)
		require.Equal(t, "updated-value", updatedDomain.Spec["test-string"])
		require.Equal(t, int64(84), updatedDomain.Spec["test-number"])
		require.Nil(t, updatedDomain.Spec["test-bool"])
		require.Nil(t, updatedDomain.Spec["test-list"])
		require.Nil(t, updatedDomain.Spec["test-map"])
	})

	t.Run("delete_workspace", func(t *testing.T) {
		// Delete the workspace
		metadata := regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: workspaceName,
			},
			Scope: scope.Scope{
				Tenant: tenant,
			},
		}

		err := deleteController.Do(ctx, &metadata)
		require.NoError(t, err)

		// Verify deletion
		_, err = getController.Do(ctx, &metadata)
		require.Error(t, err)
		require.ErrorContains(t, err, "not found")
	})
}
