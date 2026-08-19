package kubeclient

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
)

// ClientFlags holds kube client rate-limit settings bound to cobra flags.
type ClientFlags struct {
	// QPS is the steady client-go request rate to the apiserver.
	// Values less than 0 disable client-side rate limiting.
	QPS float32
	// Burst is the maximum number of requests that may exceed QPS for a short time.
	// Must be at least 1 when QPS is greater than 0.
	Burst int
}

// RegisterClientFlags adds kube client rate-limit flags to the given command.
func RegisterClientFlags(cmd *cobra.Command, f *ClientFlags) {
	cmd.Flags().Float32Var(
		&f.QPS,
		"kube-qps",
		DefaultQPS,
		"Maximum queries per second from this process to the Kubernetes apiserver "+
			"(client-go token bucket). Set to a negative value to disable client-side rate limiting",
	)
	cmd.Flags().IntVar(
		&f.Burst,
		"kube-burst",
		DefaultBurst,
		"Maximum burst size for the Kubernetes client rate limiter "+
			"(must be >= 1 when --kube-qps > 0)",
	)
}

// ApplyToConfig writes QPS and Burst onto cfg after validation.
// Call this before kubeclient.NewFromConfig so the values are on rest.Config
// when the client is built.
func (f *ClientFlags) ApplyToConfig(cfg *rest.Config) error {
	if f == nil {
		return fmt.Errorf("client flags cannot be nil")
	}
	if cfg == nil {
		return fmt.Errorf("rest.Config cannot be nil")
	}
	if f.QPS > 0 && f.Burst < 1 {
		return fmt.Errorf(
			"kube-burst must be >= 1 when kube-qps > 0 (got kube-qps=%v kube-burst=%d)",
			f.QPS, f.Burst,
		)
	}
	cfg.QPS = f.QPS
	cfg.Burst = f.Burst
	return nil
}
