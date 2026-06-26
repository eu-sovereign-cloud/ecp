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

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	resource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	rolek8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role/backend/kubernetes"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
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
	workspaceRepo    persistence.Repo[*wsdom.Workspace]
	blockStorageRepo persistence.Repo[*bsdom.BlockStorage]
	imageRepo        persistence.Repo[*imgdom.Image]
	roleRepo         persistence.Repo[*roledom.Role]
	k8sClient        client.Client
)

func TestMain(m *testing.M) {
	// Initialize k8s scheme for client-go
	s := runtime.NewScheme()
	utilruntime.Must(scheme.AddToScheme(s))
	utilruntime.Must(rolek8s.AddToScheme(s))
	utilruntime.Must(wsk8s.AddToScheme(s))
	utilruntime.Must(bsk8s.AddToScheme(s))
	utilruntime.Must(imgk8s.AddToScheme(s))
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

	// Initialize dynamic clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
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
	blockStorageRepo = k8sadapter.NewRepoAdapter(
		dynamicClient,
		bsk8s.BlockStorageGVR,
		testLogger,
		bsk8s.BlockStorageToCR,
		bsk8s.BlockStorageFromCR,
	)

	imageRepo = k8sadapter.NewRepoAdapter(
		dynamicClient,
		imgk8s.ImageGVR,
		testLogger,
		imgk8s.ImageToCR,
		imgk8s.ImageFromCR,
	)

	roleRepo = k8sadapter.NewRepoAdapter(
		dynamicClient,
		rolek8s.RoleGVR,
		testLogger,
		rolek8s.RoleToCR,
		rolek8s.RoleFromCR,
	)

	workspaceRepo = k8sadapter.NewNamespaceManagingRepoAdapter(
		dynamicClient,
		clientset,
		wsk8s.WorkspaceGVR,
		testLogger,
		wsk8s.WorkspaceToCR,
		wsk8s.WorkspaceFromCR,
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

func createTestWorkspace(ctx context.Context, workspaceRepo persistence.Repo[*wsdom.Workspace]) error {
	wsDomain := &wsdom.Workspace{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{
				Name: testWorkspace,
			},
			Scope: resource.Scope{
				Tenant: testTenant,
			},
		},
	}

	_, err := workspaceRepo.Create(ctx, wsDomain)

	return err
}

func cleanupTestWorkspace(ctx context.Context, workspaceRepo persistence.Repo[*wsdom.Workspace]) error {
	wsDomain := &wsdom.Workspace{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{
				Name: testWorkspace,
			},
			Scope: resource.Scope{
				Tenant: testTenant,
			},
		},
	}

	return workspaceRepo.Delete(ctx, wsDomain)
}
