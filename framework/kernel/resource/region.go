package resource

import "context"

// regionContextKey is the unexported context key for the region a request is
// addressed to. A private type prevents collisions with other packages' keys.
type regionContextKey struct{}

// ContextWithRegion returns ctx carrying the region this request is addressed to.
//
// A gateway process may serve several regions at once (selected per request by URL
// path segment or by host), so the region is request-scoped rather than a property
// of the process. It is set once, at the edge, by the region router middleware.
func ContextWithRegion(ctx context.Context, region string) context.Context {
	return context.WithValue(ctx, regionContextKey{}, region)
}

// RegionFromContext returns the region stored by ContextWithRegion, or fallback when
// none is set — the global gateway (no region at all), a handler called directly by a
// unit test, and any read path that does not originate from an HTTP request.
func RegionFromContext(ctx context.Context, fallback string) string {
	if region, ok := ctx.Value(regionContextKey{}).(string); ok && region != "" {
		return region
	}
	return fallback
}
