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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
)

const (
	testNamespace = "e2e-ecp"
	serviceName   = "gateway-global-svc"
	pollInterval  = 5 * time.Second
	timeout       = 2 * time.Minute
)

var (
	testLogger   *slog.Logger
	regionClient *regionv1.ClientWithResponses
)

func TestMain(m *testing.M) {
	// Initialize kubeconfig and clientset
	restConfig, clientset, err := setupK8sClient()
	if err != nil {
		log.Fatalf("Failed to set up k8s client: %v", err)
	}

	// Wait for the service to exist
	if err := waitForService(clientset, serviceName, testNamespace); err != nil {
		log.Fatalf("Error waiting for service %s: %v", serviceName, err)
	}

	// Set up port forwarding
	stopCh := make(chan struct{})
	defer close(stopCh)
	readyCh := make(chan struct{})

	pf, err := setupPortForward(clientset, restConfig, stopCh, readyCh)
	if err != nil {
		log.Fatalf("Failed to setup port forwarding: %v", err)
	}

	go func() {
		if err := pf.ForwardPorts(); err != nil {
			// Don't log fatal here as this can happen during cleanup
			log.Printf("Port forwarding failed: %v", err)
		}
	}()

	select {
	case <-readyCh:
		log.Println("Port forwarding is ready.")
	case <-time.After(timeout):
		log.Fatalf("Timed out waiting for port forwarding to become ready")
	}

	forwardedPorts, err := pf.GetPorts()
	if err != nil || len(forwardedPorts) == 0 {
		log.Fatalf("Failed to get forwarded ports: %v", err)
	}
	localPort := forwardedPorts[0].Local

	// Initialize SDK client
	serverURL := fmt.Sprintf("http://localhost:%d/providers/seca.region", localPort)
	regionClient, err = regionv1.NewClientWithResponses(serverURL)
	if err != nil {
		log.Fatalf("Failed to create region SDK client: %v", err)
	}

	// Initialize test logger
	testLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Run tests
	log.Println("Test environment ready. Running tests...")
	code := m.Run()

	os.Exit(code)
}

func setupK8sClient() (*rest.Config, *kubernetes.Clientset, error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
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

func waitForService(clientset *kubernetes.Clientset, name, namespace string) error {
	log.Printf("Waiting for service %s in namespace %s...", name, namespace)
	return wait.PollUntilContextTimeout(context.Background(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		_, err := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil // Keep trying
		}
		log.Printf("Service %s found.", name)
		return true, nil
	})
}

func setupPortForward(clientset *kubernetes.Clientset, config *rest.Config, stopCh, readyCh chan struct{}) (*portforward.PortForwarder, error) {
	var podName string
	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		pods, err := clientset.CoreV1().Pods(testNamespace).List(ctx, metav1.ListOptions{LabelSelector: "app=gateway-global"})
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
		return nil, fmt.Errorf("failed to find a running pod for gateway-global: %w", err)
	}

	log.Printf("Found pod %s to port-forward to.", podName)

	reqURL, err := url.Parse(fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/portforward", config.Host, testNamespace, podName))
	if err != nil {
		return nil, err
	}

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, reqURL)
	// Using port 0 allows the OS to choose a random available local port.
	ports := []string{"0:8080"}

	return portforward.New(dialer, ports, stopCh, readyCh, io.Discard, io.Discard)
}
