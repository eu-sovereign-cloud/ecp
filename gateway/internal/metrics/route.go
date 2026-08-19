package metrics

import (
	"net/http"
	"strings"
)

// routeOther is the catch-all route label. Every SECA route is registered on the
// shared http.ServeMux with a method+template pattern, so a matched request always
// carries r.Pattern; the fallback below only ever sees unmatched paths. Those are
// caller-controlled and unauthenticated, so they must never contribute a label
// value derived from the request beyond the closed sets in this file.
const routeOther = "{other}"

// secaProviders is the closed set of providers this gateway serves (the BaseURLs
// registered in cmd/globalapiserver.go and cmd/regionalapiserver.go, mirroring
// go-sdk's pkg/constants provider names). Used as an allowlist for the provider
// label so an arbitrary "seca.<random>" segment cannot mint a new series.
var secaProviders = map[string]struct{}{
	"seca.authorization": {},
	"seca.region":        {},
	"seca.workspace":     {},
	"seca.storage":       {},
	"seca.compute":       {},
	"seca.network":       {},
}

// apiVersions is the closed set of API versions that may appear after the
// provider. Matched exactly — a prefix test would accept "v1<random>".
var apiVersions = map[string]struct{}{
	"v1":      {},
	"v1beta1": {},
}

// systemPaths are served outside the provider namespace.
var systemPaths = map[string]struct{}{
	"/healthz": {},
	"/readyz":  {},
	"/metrics": {},
}

// routeFromRequest returns a low-cardinality route template and provider label.
// Prefer r.Pattern (set by http.ServeMux after a match); fall back to
// normalizePath so unmatched requests still have bounded cardinality.
func routeFromRequest(r *http.Request) (route, provider string) {
	if pattern := r.Pattern; pattern != "" {
		route = stripMethodPrefix(pattern)
		return route, providerFromPath(route)
	}
	return normalizePath(r.URL.Path)
}

// stripMethodPrefix removes a leading "METHOD " from a ServeMux pattern such as
// "GET /providers/seca.workspace/v1/tenants/{tenant}/workspaces".
func stripMethodPrefix(pattern string) string {
	method, path, ok := strings.Cut(pattern, " ")
	if !ok {
		return pattern
	}
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPost, http.MethodPatch,
		http.MethodDelete, http.MethodHead, http.MethodOptions:
		return path
	default:
		return pattern
	}
}

// normalizePath maps an unmatched request path to a bounded (route, provider)
// pair. Only literals from the closed allowlists above are preserved: the rest of
// the path collapses to {other}, so the label set is at most
// len(secaProviders)*len(apiVersions)+1 routes plus the system paths.
func normalizePath(path string) (route, provider string) {
	if _, ok := systemPaths[path]; ok {
		return path, "system"
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || parts[0] != "providers" {
		return routeOther, "unknown"
	}

	provider = parts[1]
	if _, ok := secaProviders[provider]; !ok {
		return routeOther, "unknown"
	}
	if len(parts) < 3 {
		return routeOther, provider
	}
	if _, ok := apiVersions[parts[2]]; !ok {
		return routeOther, provider
	}

	return "/providers/" + provider + "/" + parts[2] + "/" + routeOther, provider
}

// providerFromPath extracts an allowlisted provider from a route template or path.
func providerFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "providers" {
		if _, ok := secaProviders[parts[1]]; ok {
			return parts[1]
		}
	}
	if _, ok := systemPaths[path]; ok {
		return "system"
	}
	return "unknown"
}
