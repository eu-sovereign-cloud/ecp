package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	kresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

const (
	regionOne = "region-one"
	regionTwo = "region-two"
)

// spyHandler records what the mux downstream of the region router would have seen.
type spyHandler struct {
	called bool
	path   string
	region string
}

func (s *spyHandler) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	s.called = true
	s.path = r.URL.Path
	s.region = kresource.RegionFromContext(r.Context(), "")
}

func serveRegion(t *testing.T, regions []string, defaultRegion, target, host string) (*httptest.ResponseRecorder, *spyHandler) {
	t.Helper()
	spy := &spyHandler{}
	h := NewRegionRouter(regions, defaultRegion, slog.New(slog.NewTextHandler(io.Discard, nil)))(spy)
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	if host != "" {
		r.Host = host
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w, spy
}

func TestRegionRouter(t *testing.T) {
	t.Parallel()
	const providerPath = "/providers/seca.workspace/v1/tenants/t1/workspaces"

	t.Run("path prefix selects the region and is stripped before routing", func(t *testing.T) {
		_, spy := serveRegion(t, []string{regionOne, regionTwo}, regionOne, RegionPathPrefix+regionTwo+providerPath, "")
		require.True(t, spy.called)
		require.Equal(t, regionTwo, spy.region)
		require.Equal(t, providerPath, spy.path, "the mux must match the same pattern as an unprefixed request")
	})

	t.Run("an unserved region is a 404, not a fallback", func(t *testing.T) {
		w, spy := serveRegion(t, []string{regionOne}, regionOne, RegionPathPrefix+"elsewhere"+providerPath, "")
		require.False(t, spy.called)
		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("host first label selects the region", func(t *testing.T) {
		_, spy := serveRegion(t, []string{regionOne, regionTwo}, regionOne, providerPath, regionTwo+".api.example.com:8080")
		require.Equal(t, regionTwo, spy.region)
		require.Equal(t, providerPath, spy.path)
	})

	t.Run("an unrelated host falls back to the default region", func(t *testing.T) {
		_, spy := serveRegion(t, []string{regionOne, regionTwo}, regionOne, providerPath, "gateway.example.com")
		require.Equal(t, regionOne, spy.region)
	})

	t.Run("path wins over host", func(t *testing.T) {
		_, spy := serveRegion(t, []string{regionOne, regionTwo}, regionOne, RegionPathPrefix+regionOne+providerPath, regionTwo+".example.com")
		require.Equal(t, regionOne, spy.region)
	})

	t.Run("several regions and no default rejects a provider request naming none", func(t *testing.T) {
		w, spy := serveRegion(t, []string{regionOne, regionTwo}, "", providerPath, "")
		require.False(t, spy.called)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("probes are served whether or not a region was named", func(t *testing.T) {
		_, spy := serveRegion(t, []string{regionOne, regionTwo}, "", "/readyz", "")
		require.True(t, spy.called)
		require.Equal(t, "/readyz", spy.path)
	})

	t.Run("a single region behaves exactly as before", func(t *testing.T) {
		_, spy := serveRegion(t, []string{regionOne}, regionOne, providerPath, "")
		require.Equal(t, regionOne, spy.region)
		require.Equal(t, providerPath, spy.path)
	})

	t.Run("an escaped path segment survives the strip", func(t *testing.T) {
		_, spy := serveRegion(t, []string{regionOne}, regionOne, RegionPathPrefix+regionOne+"/providers/seca.workspace/v1/tenants/a%20b/workspaces", "")
		require.Equal(t, "/providers/seca.workspace/v1/tenants/a b/workspaces", spy.path)
	})
}

// TestRegionRouter_ServesTheRoutedMux pins the property the strip exists for: the same
// route registration answers a prefixed and an unprefixed request, and r.Pattern — which
// the authorization claim extractor and the metrics route label both read — is the
// unprefixed pattern either way.
func TestRegionRouter_ServesTheRoutedMux(t *testing.T) {
	t.Parallel()
	const pattern = "GET /providers/seca.workspace/v1/tenants/{tenant}/workspaces"
	var gotPattern, gotTenant, gotRegion string
	mux := http.NewServeMux()
	mux.HandleFunc(pattern, func(_ http.ResponseWriter, r *http.Request) {
		gotPattern, gotTenant = r.Pattern, r.PathValue("tenant")
		gotRegion = kresource.RegionFromContext(r.Context(), "")
	})
	h := NewRegionRouter([]string{regionOne, regionTwo}, regionOne, slog.New(slog.NewTextHandler(io.Discard, nil)))(mux)

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, RegionPathPrefix+regionTwo+"/providers/seca.workspace/v1/tenants/t1/workspaces", nil)
	h.ServeHTTP(httptest.NewRecorder(), r)

	require.Equal(t, pattern, gotPattern)
	require.Equal(t, "t1", gotTenant)
	require.Equal(t, regionTwo, gotRegion)
}
