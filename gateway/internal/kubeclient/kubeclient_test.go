package kubeclient_test

import (
	"testing"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"

	"github.com/eu-sovereign-cloud/ecp/gateway/internal/kubeclient"
)

func TestNewFromConfig_AppliesDefaultsWhenUnset(t *testing.T) {
	t.Parallel()

	cfg := &rest.Config{Host: "https://127.0.0.1:6443"}
	client, err := kubeclient.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	if client == nil || client.Client == nil || client.ClientSet == nil {
		t.Fatal("expected non-nil clients")
	}

	if cfg.QPS != kubeclient.DefaultQPS {
		t.Errorf("QPS = %v, want %v", cfg.QPS, kubeclient.DefaultQPS)
	}
	if cfg.Burst != kubeclient.DefaultBurst {
		t.Errorf("Burst = %v, want %v", cfg.Burst, kubeclient.DefaultBurst)
	}
	if cfg.UserAgent != kubeclient.DefaultUserAgent {
		t.Errorf("UserAgent = %q, want %q", cfg.UserAgent, kubeclient.DefaultUserAgent)
	}
	if cfg.RateLimiter == nil {
		t.Fatal("expected shared RateLimiter when QPS > 0")
	}
}

func TestNewFromConfig_PreservesExplicitOverrides(t *testing.T) {
	t.Parallel()

	cfg := &rest.Config{
		Host:      "https://127.0.0.1:6443",
		QPS:       42,
		Burst:     84,
		UserAgent: "custom-agent",
	}
	if _, err := kubeclient.NewFromConfig(cfg); err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}

	if cfg.QPS != 42 {
		t.Errorf("QPS = %v, want 42", cfg.QPS)
	}
	if cfg.Burst != 84 {
		t.Errorf("Burst = %v, want 84", cfg.Burst)
	}
	if cfg.UserAgent != "custom-agent" {
		t.Errorf("UserAgent = %q, want custom-agent", cfg.UserAgent)
	}
	if cfg.RateLimiter == nil {
		t.Fatal("expected shared RateLimiter when QPS > 0")
	}
}

func TestNewFromConfig_NilConfig(t *testing.T) {
	t.Parallel()

	_, err := kubeclient.NewFromConfig(nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestNewFromConfig_PartialOverrides(t *testing.T) {
	t.Parallel()

	// Only QPS set: Burst and UserAgent still get defaults.
	cfg := &rest.Config{
		Host: "https://127.0.0.1:6443",
		QPS:  25,
	}
	if _, err := kubeclient.NewFromConfig(cfg); err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	if cfg.QPS != 25 {
		t.Errorf("QPS = %v, want 25", cfg.QPS)
	}
	if cfg.Burst != kubeclient.DefaultBurst {
		t.Errorf("Burst = %v, want %v", cfg.Burst, kubeclient.DefaultBurst)
	}
	if cfg.UserAgent != kubeclient.DefaultUserAgent {
		t.Errorf("UserAgent = %q, want %q", cfg.UserAgent, kubeclient.DefaultUserAgent)
	}
}

func TestNewFromConfig_SharesRateLimiterWithTypedClient(t *testing.T) {
	t.Parallel()

	cfg := &rest.Config{
		Host:  "https://127.0.0.1:6443",
		QPS:   50,
		Burst: 100,
	}
	client, err := kubeclient.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	if cfg.RateLimiter == nil {
		t.Fatal("expected RateLimiter on config")
	}

	// Typed discovery REST client must use the same limiter instance.
	got := client.ClientSet.Discovery().RESTClient().GetRateLimiter()
	if got == nil {
		t.Fatal("expected non-nil rate limiter on discovery REST client")
	}
	if got != cfg.RateLimiter {
		t.Fatal("typed client RateLimiter is not the shared config RateLimiter")
	}
}

func TestNewFromConfig_PreservesCallerRateLimiter(t *testing.T) {
	t.Parallel()

	custom := flowcontrol.NewTokenBucketRateLimiter(11, 12)
	cfg := &rest.Config{
		Host:        "https://127.0.0.1:6443",
		QPS:         5,
		Burst:       10,
		RateLimiter: custom,
	}
	client, err := kubeclient.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	if cfg.RateLimiter != custom {
		t.Fatal("caller RateLimiter was replaced")
	}
	got := client.ClientSet.Discovery().RESTClient().GetRateLimiter()
	if got != custom {
		t.Fatal("typed client does not use caller RateLimiter")
	}
}

func TestNewFromConfig_DisablesRateLimitWhenQPSNegative(t *testing.T) {
	t.Parallel()

	cfg := &rest.Config{
		Host:  "https://127.0.0.1:6443",
		QPS:   -1,
		Burst: 0,
	}
	client, err := kubeclient.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if cfg.RateLimiter != nil {
		t.Fatal("expected no RateLimiter when QPS < 0")
	}
	if got := client.ClientSet.Discovery().RESTClient().GetRateLimiter(); got != nil {
		t.Fatal("expected no rate limiter on discovery REST client when QPS < 0")
	}
}

func TestNewFromConfig_RejectsInvalidBurst(t *testing.T) {
	t.Parallel()

	// Negative burst is not filled by defaults (only zero is).
	cfg := &rest.Config{
		Host:  "https://127.0.0.1:6443",
		QPS:   10,
		Burst: -1,
	}
	if _, err := kubeclient.NewFromConfig(cfg); err == nil {
		t.Fatal("expected error when QPS > 0 and Burst < 1")
	}
}
