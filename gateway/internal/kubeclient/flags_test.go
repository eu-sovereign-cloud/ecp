package kubeclient_test

import (
	"testing"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"

	"github.com/eu-sovereign-cloud/ecp/gateway/internal/kubeclient"
)

func TestRegisterClientFlags_Defaults(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	var flags kubeclient.ClientFlags
	kubeclient.RegisterClientFlags(cmd, &flags)

	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if flags.QPS != kubeclient.DefaultQPS {
		t.Errorf("QPS default = %v, want %v", flags.QPS, kubeclient.DefaultQPS)
	}
	if flags.Burst != kubeclient.DefaultBurst {
		t.Errorf("Burst default = %v, want %v", flags.Burst, kubeclient.DefaultBurst)
	}
}

func TestRegisterClientFlags_Overrides(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	var flags kubeclient.ClientFlags
	kubeclient.RegisterClientFlags(cmd, &flags)

	if err := cmd.ParseFlags([]string{"--kube-qps=50", "--kube-burst=100"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if flags.QPS != 50 {
		t.Errorf("QPS = %v, want 50", flags.QPS)
	}
	if flags.Burst != 100 {
		t.Errorf("Burst = %v, want 100", flags.Burst)
	}
}

func TestClientFlags_ApplyToConfig(t *testing.T) {
	t.Parallel()

	flags := kubeclient.ClientFlags{QPS: 42, Burst: 84}
	cfg := &rest.Config{Host: "https://127.0.0.1:6443"}
	if err := flags.ApplyToConfig(cfg); err != nil {
		t.Fatalf("ApplyToConfig: %v", err)
	}
	if cfg.QPS != 42 {
		t.Errorf("QPS = %v, want 42", cfg.QPS)
	}
	if cfg.Burst != 84 {
		t.Errorf("Burst = %v, want 84", cfg.Burst)
	}
}

func TestClientFlags_ApplyToConfig_RejectsInvalidBurst(t *testing.T) {
	t.Parallel()

	flags := kubeclient.ClientFlags{QPS: 10, Burst: 0}
	cfg := &rest.Config{Host: "https://127.0.0.1:6443"}
	if err := flags.ApplyToConfig(cfg); err == nil {
		t.Fatal("expected error when QPS > 0 and Burst < 1")
	}
}

func TestClientFlags_ApplyToConfig_AllowsDisabledRateLimit(t *testing.T) {
	t.Parallel()

	// Negative QPS disables client-side limiting; burst is not required.
	flags := kubeclient.ClientFlags{QPS: -1, Burst: 0}
	cfg := &rest.Config{Host: "https://127.0.0.1:6443"}
	if err := flags.ApplyToConfig(cfg); err != nil {
		t.Fatalf("ApplyToConfig: %v", err)
	}
	if cfg.QPS != -1 {
		t.Errorf("QPS = %v, want -1", cfg.QPS)
	}
	if cfg.Burst != 0 {
		t.Errorf("Burst = %v, want 0", cfg.Burst)
	}
}

func TestClientFlags_ApplyToConfig_NilArgs(t *testing.T) {
	t.Parallel()

	var flags *kubeclient.ClientFlags
	if err := flags.ApplyToConfig(&rest.Config{}); err == nil {
		t.Fatal("expected error for nil ClientFlags")
	}

	f := kubeclient.ClientFlags{QPS: 5, Burst: 10}
	if err := f.ApplyToConfig(nil); err == nil {
		t.Fatal("expected error for nil rest.Config")
	}
}

func TestNewFromConfig_UsesAppliedFlags(t *testing.T) {
	t.Parallel()

	cfg := &rest.Config{Host: "https://127.0.0.1:6443"}
	flags := kubeclient.ClientFlags{QPS: 50, Burst: 100}
	if err := flags.ApplyToConfig(cfg); err != nil {
		t.Fatalf("ApplyToConfig: %v", err)
	}

	client, err := kubeclient.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	if client == nil || client.Client == nil || client.ClientSet == nil {
		t.Fatal("expected non-nil clients")
	}
	if cfg.QPS != 50 {
		t.Errorf("QPS = %v, want 50", cfg.QPS)
	}
	if cfg.Burst != 100 {
		t.Errorf("Burst = %v, want 100", cfg.Burst)
	}
}
