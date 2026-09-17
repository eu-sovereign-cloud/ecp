package middleware

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"

	rest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	kresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

// RegionPathPrefix anchors the per-request region segment: a request addressed to
// "/regions/<region>/providers/…" is served as that region, and the prefix is stripped
// before routing so every provider route is registered exactly once.
const RegionPathPrefix = "/regions/"

// providerPathPrefix is where the SECA provider routes are mounted. Only these routes
// need a region; probes and /metrics are served whatever the request resolves to.
const providerPathPrefix = "/providers/"

// NewRegionRouter returns a middleware that resolves which of the regions this process
// serves a request is addressed to, and stores it in the request context
// ([kresource.ContextWithRegion]) for the handlers, the authorization claim extractor and
// the region-scoped read adapters downstream.
//
// Resolution order, first match wins:
//
//  1. Path — "/regions/<region>/providers/…". The prefix is stripped before the request
//     reaches the mux, so the provider routes, r.Pattern and the metrics route label are
//     identical whether or not a region was named. A region the process does not serve is
//     a 404: silently falling back would place the resource in the wrong region.
//  2. Host — the first DNS label of the Host header ("itbg-bergamo.api.example.com"),
//     when it names a served region. This is the domain-routed deployment.
//  3. defaultRegion — the single region the process was configured with (--region).
//     When it is empty (several regions and no default) a provider request that named
//     no region is a 400 rather than a guess.
//
// Wrap the whole mux with it, outside the metrics middleware, so the path is already
// stripped when the mux matches and r.Pattern is set on the request metrics observes.
func NewRegionRouter(regions []string, defaultRegion string, log *slog.Logger) func(http.Handler) http.Handler {
	served := make(map[string]struct{}, len(regions))
	for _, r := range regions {
		served[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if region, tail, ok := splitRegionPrefix(r.URL.EscapedPath()); ok {
				if _, isServed := served[region]; !isServed {
					log.WarnContext(r.Context(), "request for a region this gateway does not serve",
						slog.String("region", region))
					rest.WriteErrorResponse(w, r, log,
						kernel.NewError(kernel.KindNotFound, fmt.Errorf("region %q is not served by this gateway", region)))
					return
				}
				next.ServeHTTP(w, withRegion(r, region, tail))
				return
			}

			region := defaultRegion
			if fromHost := hostRegion(r.Host, served); fromHost != "" {
				region = fromHost
			}
			if region == "" && strings.HasPrefix(r.URL.Path, providerPathPrefix) {
				rest.WriteErrorResponse(w, r, log, fmt.Errorf(
					"this gateway serves several regions and has no default: address the request to %s<region>%s…: %w",
					RegionPathPrefix, providerPathPrefix, rest.ErrBadRequest))
				return
			}
			next.ServeHTTP(w, r.WithContext(kresource.ContextWithRegion(r.Context(), region)))
		})
	}
}

// splitRegionPrefix splits an escaped request path of the form
// "/regions/<region>/<tail>" into the unescaped region and the still-escaped tail.
// ok is false when the path carries no region segment.
func splitRegionPrefix(escapedPath string) (region, tail string, ok bool) {
	if !strings.HasPrefix(escapedPath, RegionPathPrefix) {
		return "", "", false
	}
	seg := escapedPath[len(RegionPathPrefix):]
	tail = "/"
	if i := strings.IndexByte(seg, '/'); i >= 0 {
		seg, tail = seg[:i], seg[i:]
	}
	if seg == "" {
		return "", "", false
	}
	unescaped, err := url.PathUnescape(seg)
	if err != nil {
		return "", "", false
	}
	return unescaped, tail, true
}

// withRegion returns a shallow copy of r whose context carries the region and whose path
// is tail — the request as it would have arrived without the region prefix.
func withRegion(r *http.Request, region, tail string) *http.Request {
	r2 := r.WithContext(kresource.ContextWithRegion(r.Context(), region))
	u := *r2.URL
	u.RawPath = tail
	path, err := url.PathUnescape(tail)
	if err != nil {
		path = tail
	}
	u.Path = path
	r2.URL = &u
	return r2
}

// hostRegion returns the served region named by the first DNS label of host, or "".
func hostRegion(host string, served map[string]struct{}) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	label, _, _ := strings.Cut(host, ".")
	if _, ok := served[label]; ok {
		return label
	}
	return ""
}
