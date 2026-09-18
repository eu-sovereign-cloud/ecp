package builder_test

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
)

const offlineKubeconfig = `apiVersion: v1
kind: Config
clusters: [{name: offline, cluster: {server: "http://127.0.0.1:1"}}]
users: [{name: offline, user: {}}]
contexts: [{name: offline, context: {cluster: offline, user: offline}}]
current-context: offline
`

// offline points KUBECONFIG at an unreachable host and returns options a manager can be built
// from and started without an API server: nothing here watches a type, so no informer ever dials
// that host, and the listeners that would otherwise claim fixed ports are disabled or moved.
func offline(t *testing.T, probeAddr string) ctrl.Options {
	t.Helper()
	kubeconfig := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(kubeconfig, []byte(offlineKubeconfig), 0o600))
	t.Setenv("KUBECONFIG", kubeconfig)
	return ctrl.Options{
		HealthProbeBindAddress: probeAddr,
		Metrics:                metricsserver.Options{BindAddress: "0"},
	}
}

func TestNewDelegator_ReturnsConfigError(t *testing.T) {
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "missing"))
	t.Setenv("KUBERNETES_SERVICE_HOST", "")

	_, err := builder.NewDelegator(ctrl.Options{})
	require.Error(t, err, "a config that fails to load must reach the caller, not exit the process")
}

type recordingReconciler struct{ called atomic.Bool }

func (r *recordingReconciler) SetupWithManager(ctrl.Manager) error {
	r.called.Store(true)
	return nil
}

// scopableReconciler is a controller that can be region-scoped, as every resource slice's is
// through the generic controller it embeds.
type scopableReconciler struct {
	recordingReconciler
	regions []string
}

func (r *scopableReconciler) ScopeToRegions(regions []string) { r.regions = regions }

func TestNewDelegator_RegistersSchemes(t *testing.T) {
	widget := schema.GroupVersionKind{Group: "test.secapi.cloud", Version: "v1", Kind: "Widget"}
	opts := offline(t, "0")

	d, err := builder.NewDelegator(opts, func(s *runtime.Scheme) error {
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
	d, err := builder.NewDelegator(offline(t, addr))
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

// TestNewDelegator_RegionScope covers the one knob a regional delegator deployment is
// configured with: REGIONS reaches the controllers it restricts. The parse is forgiving on
// purpose — a trailing comma in a Helm-rendered list must not take the process down — but an
// entry it keeps becomes a watch filter, so a blank one would match only an empty region
// label, which nothing sets, and the delegator would reconcile nothing.
func TestNewDelegator_RegionScope(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want []string
	}{
		{name: "unset watches every region", env: ""},
		{name: "one region", env: "itbg-bergamo", want: []string{"itbg-bergamo"}},
		{name: "several regions", env: "itbg-bergamo,deff-frankfurt", want: []string{"itbg-bergamo", "deff-frankfurt"}},
		{name: "blanks and surrounding space are dropped", env: " itbg-bergamo , , deff-frankfurt ", want: []string{"itbg-bergamo", "deff-frankfurt"}},
		{name: "an all-blank value is no scope at all", env: " , "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("REGIONS", tc.env)
			d, err := builder.NewDelegator(offline(t, "0"))
			require.NoError(t, err)

			r := &scopableReconciler{}
			d.Controllers.Add(r)
			require.NoError(t, d.Controllers.SetupWithManager(d.Manager))
			require.Equal(t, tc.want, r.regions, "the scope has to reach the controllers, not just the Delegator")
		})
	}
}

// A plugin is free to add a Reconciler of its own that cannot be region-scoped; the set must
// still bind it, watching every region, rather than skipping it or failing the whole setup.
func TestControllerSet_UnscopableControllerIsStillSetUp(t *testing.T) {
	cs := builder.NewControllerSet("itbg-bergamo")
	r := &recordingReconciler{}
	cs.Add(r)

	require.NoError(t, cs.SetupWithManager(nil))
	require.True(t, r.called.Load())
}
