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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage"
	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/block-storages/v1"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	ecpmodel "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	regionalmodel "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	storagev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	workspacev1sdk "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
)

const (
	pollInterval    = 5 * time.Second
	timeout         = 3 * time.Minute
	systemNamespace = "e2e-ecp"
	testTenant      = "test-tenant"
	testWorkspace   = "test-workspace"
)

var (
	k8sClient        client.Client
	clientset        *kubernetes.Clientset
	dynamicClient    dynamic.Interface
	testLogger       *slog.Logger
	regionClient     *regionv1.ClientWithResponses
	storageClient    *storagev1.ClientWithResponses
	workspaceClient  *workspacev1sdk.ClientWithResponses
	workspaceRepo    port.Repo[*regionalmodel.WorkspaceDomain]
	blockStorageRepo port.Repo[*regionalmodel.BlockStorageDomain]
)

func TestMain(m *testing.M) {
	var err error

	// Kubernetes clients setup
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

	dynamicClient, err = dynamic.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
	}

	// SECA repos setup
	workspaceRepo = kubernetesadapter.NewNamespaceManagingRepoAdapter(
		dynamicClient,
		clientset,
		workspacev1.WorkspaceGVR,
		testLogger,
		kubernetesadapter.MapWorkspaceDomainToCR,
		kubernetesadapter.MapCRToWorkspaceDomain,
	)

	blockStorageRepo = kubernetesadapter.NewRepoAdapter(
		dynamicClient,
		blockstoragev1.BlockStorageGVR,
		testLogger,
		kubernetesadapter.MapBlockStorageDomainToCR,
		kubernetesadapter.MapCRToBlockStorageDomain,
	)

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

	// SECA SDK clients setup
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
		pods, err := clientset.CoreV1().Pods(systemNamespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
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
	reqURL, err := url.Parse(fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/portforward", config.Host, systemNamespace, podName))
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

func createTestWorkspace(ctx context.Context, workspaceRepo port.Repo[*regionalmodel.WorkspaceDomain]) error {
	wsDomain := &regionalmodel.WorkspaceDomain{
		Metadata: regionalmodel.Metadata{
			CommonMetadata: ecpmodel.CommonMetadata{
				Name: testWorkspace,
			},
			Scope: scope.Scope{
				Tenant: testTenant,
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
				Tenant: testTenant,
			},
		},
		Spec: regionalmodel.WorkspaceSpec{},
	}

	return workspaceRepo.Delete(ctx, wsDomain)
}
