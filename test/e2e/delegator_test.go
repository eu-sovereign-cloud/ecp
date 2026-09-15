//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/ecp/test/internal/testenv"
)

// TestDelegatorProbes asserts the running delegator answers the liveness and readiness probes
// charts/delegator points at it. Every plugin gets them from the shared bootstrap
// (framework/backend/kubernetes/builder.NewDelegator) rather than its own main, and helm's
// --wait only observes readiness once, at deploy time.
func TestDelegatorProbes(t *testing.T) {
	restConfig, clientset, err := testenv.SetupK8sClient()
	require.NoError(t, err)

	pf, err := testenv.StartPortForwardTo(clientset, restConfig, systemNamespace, "app=delegator", 8081)
	require.NoError(t, err)
	t.Cleanup(pf.Close)

	for _, path := range []string{"/healthz", "/readyz"} {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", pf.LocalPort, path))
		require.NoError(t, err, path)
		_ = resp.Body.Close() // the status line is all the probe reads
		assert.Equal(t, http.StatusOK, resp.StatusCode, path)
	}
}
