//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	storagev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	workspacev1sdk "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

const (
	pollInterval    = 5 * time.Second
	timeout         = 3 * time.Minute
	systemNamespace = "e2e-ecp"
	serviceName     = "gateway-regional-svc"
	testTenant      = "test-tenant"
	testWorkspace   = "test-workspace"
	// sourceBlockStorage is a workspace-scoped block storage that image tests
	// reference via "block-storages/source-bs". This suite does not wait for it to
	// reconcile — it only has to exist so the image reference resolves.
	sourceBlockStorage = "source-bs"
)

var (
	clientset       *kubernetes.Clientset
	storageClient   *storagev1.ClientWithResponses
	workspaceClient *workspacev1sdk.ClientWithResponses
)

// TestMain wires the suite to the regional gateway alone. The regional suite
// exercises the gateway's REST↔CR translation (create, read-back, update, delete)
// and never inspects reconciled status. Reconciliation to Active is the delegator's
// job and is covered by the delegator suite, so this suite needs neither the
// delegator nor the global gateway — only the regional gateway and the test-data
// fixtures (tenant namespace + storage SKUs).
func TestMain(m *testing.M) {
	restConfig, cs, err := setupK8sClient()
	if err != nil {
		log.Fatalf("Failed to set up k8s client: %v", err)
	}
	clientset = cs

	// Port forward to the regional gateway.
	regionalPort, stopPF, err := startPortForward(serviceName, "app=gateway-regional", restConfig)
	if err != nil {
		log.Fatalf("Regional gateway port-forward failed: %v", err)
	}
	defer close(stopPF)

	// SECA SDK clients, both pointing at the regional gateway.
	regionalBaseURL := fmt.Sprintf("http://localhost:%d", regionalPort)
	workspaceClient, err = workspacev1sdk.NewClientWithResponses(regionalBaseURL + "/providers/seca.workspace")
	if err != nil {
		log.Fatalf("Failed to create workspace SDK client: %v", err)
	}
	storageClient, err = storagev1.NewClientWithResponses(regionalBaseURL + "/providers/seca.storage")
	if err != nil {
		log.Fatalf("Failed to create storage SDK client: %v", err)
	}

	// Provision shared fixtures through the gateway itself. Creating the workspace
	// synchronously provisions its namespace (the gateway uses a namespace-managing
	// adapter), which the block storage tests create into; the source block storage
	// is what the image tests reference.
	if err := ensureTestWorkspace(context.Background()); err != nil {
		log.Fatalf("Failed to create test workspace: %v", err)
	}
	if err := ensureSourceBlockStorage(context.Background()); err != nil {
		log.Fatalf("Failed to create source block storage: %v", err)
	}

	exitCode := m.Run()

	// Best-effort cleanup of the shared fixtures.
	_, _ = storageClient.DeleteBlockStorageWithResponse(context.Background(), testTenant, testWorkspace, sourceBlockStorage, nil)
	_, _ = workspaceClient.DeleteWorkspaceWithResponse(context.Background(), testTenant, testWorkspace, nil)

	os.Exit(exitCode)
}

// ensureTestWorkspace creates the shared test workspace through the gateway. The
// create is an upsert, so it tolerates a leftover fixture from a previous run.
func ensureTestWorkspace(ctx context.Context) error {
	resp, err := workspaceClient.CreateOrUpdateWorkspaceWithResponse(ctx, testTenant, testWorkspace, nil, schema.Workspace{})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status %d creating workspace %q", resp.StatusCode(), testWorkspace)
	}
	return nil
}

// ensureSourceBlockStorage creates the workspace-scoped source block storage that
// image tests reference. The create is an upsert, so it tolerates a leftover fixture.
func ensureSourceBlockStorage(ctx context.Context) error {
	body := schema.BlockStorage{
		Spec: schema.BlockStorageSpec{
			SizeGB: 1,
			SkuRef: schema.Reference{Resource: "sku-1"},
		},
	}
	resp, err := storageClient.CreateOrUpdateBlockStorageWithResponse(ctx, testTenant, testWorkspace, sourceBlockStorage, nil, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status %d creating source block storage %q", resp.StatusCode(), sourceBlockStorage)
	}
	return nil
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
