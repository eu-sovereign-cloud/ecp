package builder_test

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
)

// offline returns a config and options a manager can be built from and started without an API
// server: nothing here watches a type, so no informer ever dials the unreachable host, and the
// listeners that would otherwise claim fixed ports are disabled or moved.
func offline(probeAddr string) (*rest.Config, ctrl.Options) {
	return &rest.Config{Host: "http://127.0.0.1:1"}, ctrl.Options{
		HealthProbeBindAddress: probeAddr,
		Metrics:                metricsserver.Options{BindAddress: "0"},
	}
}

type recordingReconciler struct{ called atomic.Bool }

func (r *recordingReconciler) SetupWithManager(ctrl.Manager) error {
	r.called.Store(true)
	return nil
}

func TestNewDelegator_RegistersSchemes(t *testing.T) {
	widget := schema.GroupVersionKind{Group: "test.secapi.cloud", Version: "v1", Kind: "Widget"}
	cfg, opts := offline("0")

	d, err := builder.NewDelegator(cfg, opts, func(s *runtime.Scheme) error {
		s.AddKnownTypeWithName(widget, &metav1.PartialObjectMetadata{})
		return nil
	})
	require.NoError(t, err)

	s := d.Manager.GetScheme()
	assert.True(t, s.Recognizes(corev1.SchemeGroupVersion.WithKind("Namespace")), "client-go types are always registered")
	assert.True(t, s.Recognizes(widget), "caller registrations are applied")
}

func TestDelegator_Run_ServesProbesUntilCancelled(t *testing.T) {
	addr := freeAddr(t)
	cfg, opts := offline(addr)
	d, err := builder.NewDelegator(cfg, opts)
	require.NoError(t, err)

	r := &recordingReconciler{}
	d.Controllers.Add(r)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()

	for _, path := range []string{"/healthz", "/readyz"} {
		require.Eventually(t, func() bool {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+addr+path, nil)
			require.NoError(t, err)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return false
			}
			_ = resp.Body.Close() // the status line is all the probe reads
			return resp.StatusCode == http.StatusOK
		}, 10*time.Second, 50*time.Millisecond, path)
	}
	assert.True(t, r.called.Load(), "Run must set up every controller before starting the manager")

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after its context was cancelled")
	}
}

// freeAddr reserves a loopback port and releases it for the manager to bind.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}
