package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	rest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	kresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

// RegionPathPrefix anchors the per-request region segment: a request addressed to
// "/regions/<region>/providers/…" is served as that region, and the prefix is stripped
// before routing so every provider route is registered exactly once.
const RegionPathPrefix = "/regions/"

// NewRegionRouter returns a middleware that resolves which of the regions this process
// serves a request is addressed to, and stores it in the request context
// ([kresource.ContextWithRegion]) for the handlers, the authorization claim extractor and
// the region-scoped read adapters downstream.
//
// A request names its region with a "/regions/<region>" path prefix; one that names none —
// every request to a single-region gateway, plus the probes and /metrics — is served as
// defaultRegion. A region this process does not serve is a 404: silently falling back
// would place the resource in the wrong region.
//
// The prefix is stripped ([http.StripPrefix]) before the request reaches the mux, so the
// provider routes, r.Pattern and the metrics route label are identical whether or not a
// region was named. Wrap the whole mux with it, outside the metrics middleware, so the
// path is already stripped when the mux matches and r.Pattern is set on the request
// metrics observes.
func NewRegionRouter(regions []string, defaultRegion string, log *slog.Logger) func(http.Handler) http.Handler {
	served := make(map[string]struct{}, len(regions))
	for _, r := range regions {
		served[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			region, handler := defaultRegion, next
			if tail, ok := strings.CutPrefix(r.URL.Path, RegionPathPrefix); ok {
				named, _, _ := strings.Cut(tail, "/")
				if _, isServed := served[named]; !isServed {
					log.WarnContext(r.Context(), "request for a region this gateway does not serve",
						slog.String("region", named))
					rest.WriteErrorResponse(w, r, log,
						kernel.NewError(kernel.KindNotFound, fmt.Errorf("region %q is not served by this gateway", named)))
					return
				}
				region, handler = named, http.StripPrefix(RegionPathPrefix+named, next)
			}
			handler.ServeHTTP(w, r.WithContext(kresource.ContextWithRegion(r.Context(), region)))
		})
	}
}
