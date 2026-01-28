//go:build integration

package integration

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"

	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/block-storages/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	regionalmodel "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace = "ecp-dummy-delegator"
	pollInterval  = 5 * time.Second
	timeout       = 2 * time.Minute
)

var (
	dynamicClient    dynamic.Interface
	testLogger       *slog.Logger
	workspaceRepo    *kubernetesadapter.RepoAdapter[*regionalmodel.WorkspaceDomain]
	blockStorageRepo *kubernetesadapter.RepoAdapter[*regionalmodel.BlockStorageDomain]
	k8sClient        client.Client
)

func TestMain(m *testing.M) {
	//
	// Given a running KIND cluster
	if err := setup(); err != nil {
		log.Fatalf("Failed to setup integration tests: %v", err)
	}

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
	workspaceRepo = kubernetesadapter.NewRepoAdapter(
		dynamicClient,
		workspacev1.WorkspaceGVR,
		testLogger,
		kubernetesadapter.MapWorkspaceDomainToCR,
		kubernetesadapter.MapCRToWorkspaceDomain,
	)

	// Wait for the test namespace to be created
	if err := waitForNamespace(context.Background(), testNamespace); err != nil {
		log.Fatalf("Failed to wait for namespace %s: %v", testNamespace, err)
	}

	if err := createTestNamespaces(context.Background()); err != nil {
		log.Fatalf("Failed to create test namespaces: %v", err)
	}

	//
	// When running the test suite
	code := m.Run()

	//
	// Then teardown the cluster
	if err := teardown(); err != nil {
		log.Printf("Failed to teardown integration tests: %v", err)
	}

	os.Exit(code)
}

func setup() error {
	log.Println("Setting up KIND cluster for integration tests...")
	cmd := exec.Command("make", "-C", "../../", "kind-start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func teardown() error {
	log.Println("Tearing down KIND cluster...")
	cmd := exec.Command("make", "-C", "../../", "kind-stop")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitForNamespace(ctx context.Context, namespace string) error {
	log.Printf("Waiting for namespace %s to be created...", namespace)

	return wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		var ns corev1.Namespace
		err := k8sClient.Get(ctx, client.ObjectKey{Name: namespace}, &ns)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil // Namespace not found yet, continue polling.
			}
			return false, err // Other error, stop polling.
		}
		return true, nil // Namespace found.
	})
}

func createTestNamespaces(ctx context.Context) error {
	log.Println("Creating test namespaces...")
	nsToCreate := []string{
		kubernetesadapter.ComputeNamespace(&scope.Scope{Tenant: "test-tenant"}),
		kubernetesadapter.ComputeNamespace(&scope.Scope{Tenant: "test-tenant", Workspace: "test-workspace"}),
	}

	for _, nsName := range nsToCreate {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
		if err := k8sClient.Create(ctx, ns); err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create namespace %s: %w", nsName, err)
		}
	}

	return nil
}
