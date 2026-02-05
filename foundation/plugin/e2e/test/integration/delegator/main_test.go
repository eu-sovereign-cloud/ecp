//go:build integration

package integration

import (
	"context"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage"
	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/block-storages/v1"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	ecpmodel "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	regionalmodel "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

const (
	pollInterval  = 10 * time.Second
	timeout       = 5 * time.Minute
	testTenant    = "test-tenant"
	testWorkspace = "test-workspace"
)

var (
	dynamicClient    dynamic.Interface
	testLogger       *slog.Logger
	workspaceRepo    port.Repo[*regionalmodel.WorkspaceDomain]
	blockStorageRepo port.Repo[*regionalmodel.BlockStorageDomain]
	k8sClient        client.Client
)

func TestMain(m *testing.M) {
	// Initialize k8s scheme for client-go
	s := runtime.NewScheme()
	utilruntime.Must(scheme.AddToScheme(s))
	utilruntime.Must(workspacev1.AddToScheme(s))
	utilruntime.Must(storage.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))

	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := kubeconfig.ClientConfig()
	if err != nil {
		log.Fatalf("Failed to get kubeconfig: %v", err)
	}
	restConfig.QPS = 100
	restConfig.Burst = 200

	k8sClient, err = client.New(restConfig, client.Options{Scheme: s})
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	// Initialize dynamic clientSet
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	// Initialize dynamic client
	dynamicClient, err = dynamic.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
	}

	// Initialize test logger
	testLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialize repositories
	blockStorageRepo = kubernetesadapter.NewRepoAdapter(
		dynamicClient,
		blockstoragev1.BlockStorageGVR,
		testLogger,
		kubernetesadapter.MapBlockStorageDomainToCR,
		kubernetesadapter.MapCRToBlockStorageDomain,
	)

	workspaceRepo = kubernetesadapter.NewNamespaceManagingRepoAdapter(
		dynamicClient,
		clientSet,
		workspacev1.WorkspaceGVR,
		testLogger,
		kubernetesadapter.MapWorkspaceDomainToCR,
		kubernetesadapter.MapCRToWorkspaceDomain,
	)

	// Provide Workspace for BlockStorage tests
	if err := createTestWorkspace(context.Background(), workspaceRepo); err != nil {
		log.Fatalf("Failed to create test workspace: %v", err)
	}

	// When running the test suite
	exitCode := m.Run()

	// Cleanup Workspace for BlockStorage tests
	cleanupTestWorkspace(context.Background(), workspaceRepo)

	os.Exit(exitCode)
}

func createTestWorkspace(ctx context.Context, workspaceRepo port.Repo[*regionalmodel.WorkspaceDomain]) error {
	wsDomain := &regionalmodel.WorkspaceDomain{
		Metadata: regionalmodel.Metadata{
			CommonMetadata: ecpmodel.CommonMetadata{
				Name: testWorkspace,
			},
			Scope: scope.Scope{
				Tenant:    testTenant,
				Workspace: testWorkspace,
			},
		},
		Spec: regionalmodel.WorkspaceSpec{},
	}

	_, err := workspaceRepo.Create(ctx, wsDomain)
	return err
}

func cleanupTestWorkspace(ctx context.Context, workspaceRepo port.Repo[*regionalmodel.WorkspaceDomain]) error {
	wsDomain := &regionalmodel.WorkspaceDomain{
		Metadata: regionalmodel.Metadata{
			CommonMetadata: ecpmodel.CommonMetadata{
				Name: testWorkspace,
			},
			Scope: scope.Scope{
				Tenant:    testTenant,
				Workspace: testWorkspace,
			},
		},
		Spec: regionalmodel.WorkspaceSpec{},
	}

	return workspaceRepo.Delete(ctx, wsDomain)
}
