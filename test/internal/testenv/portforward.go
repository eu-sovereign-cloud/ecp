package testenv

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	// pollInterval is how often we poll for a running pod.
	pollInterval = 2 * time.Second
	// waitTimeout bounds how long we wait for a running pod and for the
	// forwarder to become ready.
	waitTimeout = 3 * time.Minute
	// gatewayPort is the container port the ECP gateways listen on.
	gatewayPort = 8080
)

// PortForward is an active port-forward to a pod. Call Close to tear it down.
type PortForward struct {
	// LocalPort is the OS-assigned local port forwarding to the pod.
	LocalPort uint16
	stop      chan struct{}
}

// Close stops the port-forward. It is safe to call on a nil receiver.
func (pf *PortForward) Close() {
	if pf != nil && pf.stop != nil {
		close(pf.stop)
	}
}

// StartPortForward forwards a random local port to gatewayPort (8080) on the
// first running pod matching labelSelector in namespace. It blocks until the
// forwarder is ready or the wait times out.
func StartPortForward(clientset *kubernetes.Clientset, config *rest.Config, namespace, labelSelector string) (*PortForward, error) {
	stopCh := make(chan struct{})
	readyCh := make(chan struct{})

	forwarder, err := newForwarder(clientset, config, namespace, labelSelector, stopCh, readyCh)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := forwarder.ForwardPorts(); err != nil {
			// Expected during teardown; log at info so it is not mistaken for a
			// test failure.
			log.Printf("port-forward to %q in %s stopped: %v", labelSelector, namespace, err)
		}
	}()

	select {
	case <-readyCh:
		log.Printf("Port-forward to %q in %s is ready.", labelSelector, namespace)
	case <-time.After(waitTimeout):
		close(stopCh)
		return nil, fmt.Errorf("timed out waiting for port-forward to %q in %s", labelSelector, namespace)
	}

	ports, err := forwarder.GetPorts()
	if err != nil || len(ports) == 0 {
		close(stopCh)
		return nil, fmt.Errorf("failed to get forwarded ports for %q in %s: %w", labelSelector, namespace, err)
	}

	return &PortForward{LocalPort: ports[0].Local, stop: stopCh}, nil
}

// newForwarder waits for a running pod selected by labelSelector, then builds a
// SPDY-backed port-forwarder to it.
func newForwarder(clientset *kubernetes.Clientset, config *rest.Config, namespace, labelSelector string, stopCh, readyCh chan struct{}) (*portforward.PortForwarder, error) {
	var podName string
	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, waitTimeout, true, func(ctx context.Context) (bool, error) {
		pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return false, err
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning {
				podName = pod.Name
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find a running pod for %q in %s: %w", labelSelector, namespace, err)
	}

	log.Printf("Port-forwarding to pod %s (%s) in %s.", podName, labelSelector, namespace)
	reqURL, err := url.Parse(fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/portforward", config.Host, namespace, podName))
	if err != nil {
		return nil, err
	}
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, reqURL)
	// Local port 0 lets the OS choose a free port.
	ports := []string{fmt.Sprintf("0:%d", gatewayPort)}
	return portforward.New(dialer, ports, stopCh, readyCh, io.Discard, io.Discard)
}
