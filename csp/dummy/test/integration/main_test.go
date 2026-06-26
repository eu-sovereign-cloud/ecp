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

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role-assignment/v1"
	rak8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/role-assignment/v1/backend/kubernetes"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role/v1"
	rolek8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/role/v1/backend/kubernetes"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

const (
	testNamespace = "ecp-dummy-delegator"
	pollInterval  = 5 * time.Second
	timeout       = 2 * time.Minute
)

var (
	dynamicClient      dynamic.Interface
	testLogger         *slog.Logger
	networkRepo        *k8sadapter.RepoAdapter[*netdom.Network]
	workspaceRepo      *k8sadapter.RepoAdapter[*wsdom.Workspace]
	blockStorageRepo   *k8sadapter.RepoAdapter[*bsdom.BlockStorage]
	imageRepo          *k8sadapter.RepoAdapter[*imgdom.Image]
	roleRepo           *k8sadapter.RepoAdapter[*roledom.Role]
	roleAssignmentRepo *k8sadapter.RepoAdapter[*radom.RoleAssignment]
	k8sClient          client.Client
)

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		log.Fatalf("Failed to setup integration tests: %v", err)
	}

	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(rolek8s.AddToScheme(s))
	utilruntime.Must(netk8s.AddToScheme(s))
	utilruntime.Must(wsk8s.AddToScheme(s))
	utilruntime.Must(bsk8s.AddToScheme(s))
	utilruntime.Must(imgk8s.AddToScheme(s))
	utilruntime.Must(rak8s.AddToScheme(s))
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

	dynamicClient, err = dynamic.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
	}

	testLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	networkRepo = k8sadapter.NewRepoAdapter[*netdom.Network](
		dynamicClient,
		netk8s.NetworkGVR,
		testLogger,
		netk8s.NetworkToCR,
		netk8s.NetworkFromCR,
	)
	blockStorageRepo = k8sadapter.NewRepoAdapter[*bsdom.BlockStorage](
		dynamicClient,
		bsk8s.BlockStorageGVR,
		testLogger,
		bsk8s.BlockStorageToCR,
		bsk8s.BlockStorageFromCR,
	)
	workspaceRepo = k8sadapter.NewRepoAdapter[*wsdom.Workspace](
		dynamicClient,
		wsk8s.WorkspaceGVR,
		testLogger,
		wsk8s.WorkspaceToCR,
		wsk8s.WorkspaceFromCR,
	)
	imageRepo = k8sadapter.NewRepoAdapter[*imgdom.Image](
		dynamicClient,
		imgk8s.ImageGVR,
		testLogger,
		imgk8s.ImageToCR,
		imgk8s.ImageFromCR,
	)
	roleRepo = k8sadapter.NewRepoAdapter[*roledom.Role](
		dynamicClient,
		rolek8s.RoleGVR,
		testLogger,
		rolek8s.RoleToCR,
		rolek8s.RoleFromCR,
	)
	roleAssignmentRepo = k8sadapter.NewRepoAdapter[*radom.RoleAssignment](
		dynamicClient,
		rak8s.RoleAssignmentGVR,
		testLogger,
		rak8s.RoleAssignmentToCR,
		rak8s.RoleAssignmentFromCR,
	)

	if err := waitForNamespace(context.Background(), testNamespace); err != nil {
		log.Fatalf("Failed to wait for namespace %s: %v", testNamespace, err)
	}

	if err := createTestNamespaces(context.Background()); err != nil {
		log.Fatalf("Failed to create test namespaces: %v", err)
	}

	code := m.Run()

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
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func createTestNamespaces(ctx context.Context) error {
	log.Println("Creating test namespaces...")
	nsToCreate := []string{
		k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: "test-tenant"}),
		k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: "test-tenant", Workspace: "test-workspace"}),
	}

	for _, nsName := range nsToCreate {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
		if err := k8sClient.Create(ctx, ns); err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create namespace %s: %w", nsName, err)
		}
	}

	return nil
}
