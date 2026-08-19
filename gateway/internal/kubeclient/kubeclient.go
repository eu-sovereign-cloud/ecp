package kubeclient

import (
	"context"
	"fmt"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
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
//
// Dynamic and typed clients share one RateLimiter so QPS/Burst is a single
// process budget toward the apiserver. A caller-supplied RateLimiter is kept.
// QPS less than 0 disables client-side rate limiting (RateLimiter stays nil).
func NewFromConfig(cfg *rest.Config) (*KubeClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("rest.Config cannot be nil")
	}
	applyClientDefaults(cfg)
	if err := ensureSharedRateLimiter(cfg); err != nil {
		return nil, err
	}

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

// ensureSharedRateLimiter installs one token-bucket limiter on cfg when needed
// so dynamic.NewForConfig and kubernetes.NewForConfig share the same budget.
func ensureSharedRateLimiter(cfg *rest.Config) error {
	if cfg.RateLimiter != nil {
		return nil
	}
	// Negative QPS disables client-side throttling in client-go.
	if cfg.QPS < 0 {
		return nil
	}
	if cfg.Burst < 1 {
		return fmt.Errorf("burst must be >= 1 when qps > 0 (got qps=%v burst=%d)", cfg.QPS, cfg.Burst)
	}
	cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(cfg.QPS, cfg.Burst)
	return nil
}

// CheckAPIServer probes apiserver reachability for readiness (discovery /version).
func (c *KubeClient) CheckAPIServer(ctx context.Context) error {
	if c == nil || c.ClientSet == nil {
		return fmt.Errorf("kubernetes client is not configured")
	}
	return c.ClientSet.Discovery().RESTClient().Get().AbsPath("/version").Do(ctx).Error()
}
