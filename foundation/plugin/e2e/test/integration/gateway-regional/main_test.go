//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/google/uuid"

	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	storagev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	workspacev1sdk "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

const (
	testNamespace    = "e2e-ecp"
	pollInterval     = 5 * time.Second
	timeout          = 3 * time.Minute
	testTenant       = "test-tenant"
	testWorkspace    = "test-workspace"
	testWorkspaceNew = "test-workspace-new"
)

var (
	k8sClient       client.Client
	clientset       *kubernetes.Clientset
	testLogger      *slog.Logger
	regionClient    *regionv1.ClientWithResponses
	storageClient   *storagev1.ClientWithResponses
	workspaceClient *workspacev1sdk.ClientWithResponses
)

func TestMain(m *testing.M) {
	var err error
	s := runtime.NewScheme()
	utilruntime.Must(scheme.AddToScheme(s))
	utilruntime.Must(workspacev1.AddToScheme(s))
	utilruntime.Must(storage.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))

	restConfig, cs, err := setupK8sClient()
	if err != nil {
		log.Fatalf("Failed to set up k8s client: %v", err)
	}
	clientset = cs

	k8sClient, err = client.New(restConfig, client.Options{Scheme: s})
	if err != nil {
		log.Fatalf("Failed to create controller-runtime client: %v", err)
	}

	// Port forward for Global Gateway
	globalPort, stopGlobalPF, err := startPortForward("gateway-global-svc", "app=gateway-global", restConfig)
	if err != nil {
		log.Fatalf("Global gateway port-forward failed: %v", err)
	}
	defer close(stopGlobalPF)

	// Port forward for Regional Gateway
	regionalPort, stopRegionalPF, err := startPortForward("gateway-regional-svc", "app=gateway-regional", restConfig)
	if err != nil {
		log.Fatalf("Regional gateway port-forward failed: %v", err)
	}
	defer close(stopRegionalPF)

	// Initialize SDK Clients
	globalURL := fmt.Sprintf("http://localhost:%d/providers/seca.region", globalPort)
	regionClient, err = regionv1.NewClientWithResponses(globalURL)
	if err != nil {
		log.Fatalf("Failed to create region SDK client: %v", err)
	}

	regionalBaseURL := fmt.Sprintf("http://localhost:%d", regionalPort)
	workspaceClient, err = workspacev1sdk.NewClientWithResponses(regionalBaseURL + "/providers/seca.workspace")
	if err != nil {
		log.Fatalf("Failed to create workspace SDK client: %v", err)
	}
	storageClient, err = storagev1.NewClientWithResponses(regionalBaseURL + "/providers/seca.storage")
	if err != nil {
		log.Fatalf("Failed to create storage SDK client: %v", err)
	}

	testLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create namespaces for tests
	nsToCreate := []string{
		kubernetesadapter.ComputeNamespace(&scope.Scope{Tenant: testTenant}),
		kubernetesadapter.ComputeNamespace(&scope.Scope{Tenant: testTenant, Workspace: testWorkspace}),
		kubernetesadapter.ComputeNamespace(&scope.Scope{Tenant: testTenant, Workspace: testWorkspaceNew}),
		kubernetesadapter.ComputeNamespace(&scope.Scope{Tenant: testTenant, Workspace: "test-ws-create-" + uuid.New().String()[:8]}),
		kubernetesadapter.ComputeNamespace(&scope.Scope{Tenant: testTenant, Workspace: "test-ws-delete-" + uuid.New().String()[:8]}),
	}
	if err := createTestNamespaces(context.Background(), nsToCreate); err != nil {
		log.Fatalf("Failed to create test namespaces: %v", err)
	}

	log.Println("Test environment ready. Running tests...")
	code := m.Run()

	cleanupTestNamespaces(context.Background(), nsToCreate)
	os.Exit(code)
}

func startPortForward(serviceName, labelSelector string, config *rest.Config) (uint16, chan struct{}, error) {
	stopCh := make(chan struct{})
	readyCh := make(chan struct{})

	pf, err := setupPortForward(clientset, config, serviceName, labelSelector, stopCh, readyCh)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to setup port-forward for %s: %w", serviceName, err)
	}

	go func() {
		if err := pf.ForwardPorts(); err != nil {
			log.Printf("Port forwarding for %s failed: %v", serviceName, err)
		}
	}()

	select {
	case <-readyCh:
		log.Printf("Port forwarding for %s is ready.", serviceName)
	case <-time.After(timeout):
		return 0, nil, fmt.Errorf("timed out waiting for %s port-forward", serviceName)
	}

	ports, err := pf.GetPorts()
	if err != nil || len(ports) == 0 {
		return 0, nil, fmt.Errorf("failed to get forwarded ports for %s", serviceName)
	}

	return ports[0].Local, stopCh, nil
}

func setupK8sClient() (*rest.Config, *kubernetes.Clientset, error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	restConfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset: %w", err)
	}
	return restConfig, clientset, nil
}

func setupPortForward(clientset *kubernetes.Clientset, config *rest.Config, serviceName, labelSelector string, stopCh, readyCh chan struct{}) (*portforward.PortForwarder, error) {
	var podName string
	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		pods, err := clientset.CoreV1().Pods(testNamespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return false, err
		}
		if len(pods.Items) > 0 {
			for _, pod := range pods.Items {
				if pod.Status.Phase == corev1.PodRunning {
					podName = pod.Name
					return true, nil
				}
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find a running pod for %s: %w", labelSelector, err)
	}

	log.Printf("Found pod %s to port-forward to for service %s.", podName, serviceName)
	reqURL, err := url.Parse(fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/portforward", config.Host, testNamespace, podName))
	if err != nil {
		return nil, err
	}
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, reqURL)
	ports := []string{"0:8080"}
	return portforward.New(dialer, ports, stopCh, readyCh, io.Discard, io.Discard)
}

func createTestNamespaces(ctx context.Context, nsToCreate []string) error {
	log.Println("Creating test namespaces...")
	for _, nsName := range nsToCreate {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
		if err := k8sClient.Create(ctx, ns); err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create namespace %s: %w", nsName, err)
		}
	}
	return nil
}

func cleanupTestNamespaces(ctx context.Context, nsToDelete []string) {
	log.Println("Cleaning up test namespaces...")
	for _, nsName := range nsToDelete {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
		deleteCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		if err := k8sClient.Delete(deleteCtx, ns); err != nil && !kerrors.IsNotFound(err) {
			log.Printf("Failed to delete namespace %s: %v", nsName, err)
		}
	}
}
