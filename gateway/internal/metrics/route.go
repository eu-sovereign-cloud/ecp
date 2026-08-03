package metrics

import (
	"net/http"
	"strings"
)

// Fixed path segments that appear in SECA OpenAPI routes. Anything else between
// vocabulary segments is treated as a path parameter (low-cardinality template).
var routeVocabulary = map[string]struct{}{
	"providers":            {},
	"v1":                   {},
	"v1beta1":              {},
	"tenants":              {},
	"workspaces":           {},
	"regions":              {},
	"roles":                {},
	"role-assignments":     {},
	"instances":            {},
	"skus":                 {},
	"block-storages":       {},
	"images":               {},
	"networks":             {},
	"subnets":              {},
	"nics":                 {},
	"public-ips":           {},
	"route-tables":         {},
	"internet-gateways":    {},
	"security-groups":      {},
	"security-group-rules": {},
	"healthz":              {},
	"readyz":               {},
	"metrics":              {},
}

// scopePlaceholder maps a parent collection to its OpenAPI scope parameter when
// more path segments follow the value (e.g. workspaces/{workspace}/instances/...).
var scopePlaceholder = map[string]string{
	"tenants":    "{tenant}",
	"workspaces": "{workspace}",
	"networks":   "{network}",
	"clusters":   "{cluster}",
}

// routeFromRequest returns a low-cardinality route template and provider label.
// Prefer r.Pattern (set by http.ServeMux after a match); fall back to normalizing
// the URL path so unmatched or non-mux handlers still have bounded cardinality.
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

// normalizePath rewrites concrete path parameter values to placeholders.
// Unknown shapes become route "{other}" with provider "unknown".
func normalizePath(path string) (route, provider string) {
	if path == "" {
		return "{other}", "unknown"
	}
	if path == "/healthz" || path == "/readyz" {
		return path, "system"
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "{other}", "unknown"
	}

	if parts[0] != "providers" || len(parts) < 2 || !strings.HasPrefix(parts[1], "seca.") {
		return "{other}", "unknown"
	}

	provider = parts[1]
	out := make([]string, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		seg := parts[i]
		if isFixedSegment(seg) {
			out = append(out, seg)
			continue
		}
		// Variable value: choose placeholder from the preceding collection segment.
		prev := ""
		if i > 0 {
			prev = parts[i-1]
		}
		out = append(out, pathPlaceholder(prev, isLeafParam(parts, i)))
	}

	return "/" + strings.Join(out, "/"), provider
}

// isFixedSegment reports whether seg is kept literally in the route template.
func isFixedSegment(seg string) bool {
	if strings.HasPrefix(seg, "seca.") || isVersion(seg) {
		return true
	}
	_, ok := routeVocabulary[seg]
	return ok
}

// isLeafParam reports whether the path parameter at index i is terminal for its
// collection (no further vocabulary segments after it). Leaf params use {name}
// to match OpenAPI resource routes; non-leaf use scope placeholders.
func isLeafParam(parts []string, i int) bool {
	for j := i + 1; j < len(parts); j++ {
		if isFixedSegment(parts[j]) && !strings.HasPrefix(parts[j], "seca.") && !isVersion(parts[j]) {
			// A later vocabulary segment means this value is a parent scope.
			if _, ok := routeVocabulary[parts[j]]; ok {
				return false
			}
		}
	}
	return true
}

func pathPlaceholder(collection string, leaf bool) string {
	if collection == "tenants" {
		return "{tenant}"
	}
	if !leaf {
		if p, ok := scopePlaceholder[collection]; ok {
			return p
		}
	}
	return "{name}"
}

func isVersion(seg string) bool {
	return strings.HasPrefix(seg, "v1") || strings.HasPrefix(seg, "v2")
}

// providerFromPath extracts seca.<group> from a route template or path.
func providerFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "providers" && strings.HasPrefix(parts[1], "seca.") {
		return parts[1]
	}
	if path == "/healthz" || path == "/readyz" || path == "/metrics" {
		return "system"
	}
	return "unknown"
}
