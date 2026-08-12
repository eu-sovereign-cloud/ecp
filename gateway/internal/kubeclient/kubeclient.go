package kubeclient

import (
	"context"
	"fmt"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Default client-go rate limits for the gateway HTTP path. client-go defaults
// (QPS=5, Burst=10) would throttle multi-tenant parallel traffic, especially in
// the case of automation (ie Terraform) runs with large amounts of state-check
// GETs.
const (
	DefaultQPS       = 60 // Default in go-client is 5
	DefaultBurst     = 90 // Default in go-client is 10
	DefaultUserAgent = "ecp-gateway"
)

type KubeClient struct {
	Client    dynamic.Interface
	ClientSet kubernetes.Interface
}

// New loads kubeconfig and creates a KubeClient.
func New() (*KubeClient, error) {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}
	return NewFromConfig(config)
}

// NewFromConfig creates a KubeClient using the provided rest.Config.
// When QPS, Burst, or UserAgent are unset (zero/empty), gateway production
// defaults are applied. Explicit non-zero caller values are preserved.
func NewFromConfig(cfg *rest.Config) (*KubeClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("rest.Config cannot be nil")
	}
	applyClientDefaults(cfg)

	c := &KubeClient{}
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	c.Client = client
	c.ClientSet = cs
	return c, nil
}

// applyClientDefaults sets gateway QPS/Burst/UserAgent when the caller left
// them at the rest.Config zero value (client-go would otherwise use 5/10).
func applyClientDefaults(cfg *rest.Config) {
	if cfg.QPS == 0 {
		cfg.QPS = DefaultQPS
	}
	if cfg.Burst == 0 {
		cfg.Burst = DefaultBurst
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
}

// CheckAPIServer probes apiserver reachability for readiness (discovery /version).
func (c *KubeClient) CheckAPIServer(ctx context.Context) error {
	if c == nil || c.ClientSet == nil {
		return fmt.Errorf("kubernetes client is not configured")
	}
	return c.ClientSet.Discovery().RESTClient().Get().AbsPath("/version").Do(ctx).Error()
}
