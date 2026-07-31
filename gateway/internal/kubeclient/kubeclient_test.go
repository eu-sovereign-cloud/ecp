package kubeclient_test

import (
	"testing"

	"k8s.io/client-go/rest"

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
