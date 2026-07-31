// Package testenv holds setup helpers shared by the integration and e2e suites:
// loading the ambient kubeconfig and port-forwarding to in-cluster gateways.
//
// It is intentionally build-tag free so both the `integration`- and `e2e`-tagged
// suites can import it.
package testenv

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// RestConfig builds a *rest.Config from the ambient kubeconfig, honouring the
// standard loading rules (KUBECONFIG env var, --kubeconfig, ~/.kube/config).
func RestConfig() (*rest.Config, error) {
	return RestConfigForContext("")
}

// RestConfigForContext is RestConfig pinned to a named kubeconfig context, so a
// suite can hold clients for more than one cluster at once (see the multicluster
// e2e suite). An empty name falls back to the current context.
func RestConfigForContext(context string) (*rest.Config, error) {
	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{CurrentContext: context},
	)
	cfg, err := loader.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig for context %q: %w", context, err)
	}
	return cfg, nil
}

// SetupK8sClientForContext returns a *rest.Config and clientset pinned to a
// named kubeconfig context.
func SetupK8sClientForContext(context string) (*rest.Config, *kubernetes.Clientset, error) {
	cfg, err := RestConfigForContext(context)
	if err != nil {
		return nil, nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset for context %q: %w", context, err)
	}
	return cfg, clientset, nil
}

// SetupK8sClient returns the ambient *rest.Config together with a typed
// clientset built from it.
func SetupK8sClient() (*rest.Config, *kubernetes.Clientset, error) {
	cfg, err := RestConfig()
	if err != nil {
		return nil, nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset: %w", err)
	}
	return cfg, clientset, nil
}
