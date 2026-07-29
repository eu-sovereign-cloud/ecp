package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
)

// SECAClaimExtractor returns an [authzport.ClaimExtractor] that builds an
// [authzport.AuthorizationClaim] for the named SECA provider.
//
// The extractor derives the claim fields as follows:
//   - Provider: baked-in constant passed to this constructor (e.g. "seca.compute").
//   - Tenant: r.PathValue("tenant").
//   - Workspace: r.PathValue("workspace"); empty for tenant-scoped resources.
//   - Region: config.Singleton().Region(); empty on the global server.
//   - Name: r.PathValue("name"); empty for collection (list) operations.
//   - Resource: the resource kind path derived from r.Pattern (see resourceAndVerb).
//     Examples: "instances", "networks/subnets", "roles".
//   - Verb: derived from r.Method and the matched route pattern:
//     GET collection → "list", GET item → "get", PUT → "put", DELETE → "delete",
//     POST /{name}/{action} → "post.<action>".
//
// SECAClaimExtractor reads r.Pattern (available after mux routing in Go 1.22+),
// so it MUST be used after the request has been matched by the mux — which is
// guaranteed when the extractor runs inside oapi-codegen's per-route middleware chain.
func SECAClaimExtractor(provider, baseURL string) authzport.ClaimExtractor {
	return func(r *http.Request) (authzport.AuthorizationClaim, error) {
		tenant := r.PathValue("tenant")
		workspace := r.PathValue("workspace")
		name := r.PathValue("name")

		resource, verb, err := resourceAndVerb(r, baseURL, name)
		if err != nil {
			return authzport.AuthorizationClaim{}, fmt.Errorf("extract authorization claim: %w", err)
		}

		return authzport.AuthorizationClaim{
			Provider:  provider,
			Resource:  resource,
			Name:      name,
			Verb:      verb,
			Tenant:    tenant,
			Region:    config.Singleton().Region(),
			Workspace: workspace,
		}, nil
	}
}

// resourceAndVerb derives the resource kind path and RBAC verb from the matched
// HTTP request.
//
// It parses r.Pattern (e.g. "GET /providers/seca.network/v1/tenants/{tenant}/
// workspaces/{workspace}/networks/{network}/subnets/{name}") and strips the
// provider base URL and the tenant/workspace anchors. The remaining path
// segments determine the resource (only static segments, e.g. "networks/subnets")
// and the verb (from the HTTP method + whether a {name} and/or action suffix is
// present).
//
// oapi-codegen wraps POST action routes as:
//
//	POST /providers/.../instances/{name}/restart
//
// which yields verb "post.restart" and resource "instances".
func resourceAndVerb(r *http.Request, baseURL, name string) (resource, verb string, err error) {
	// r.Pattern format: "METHOD /path/pattern" — the Go 1.22+ mux sets this after matching.
	parts := strings.SplitN(r.Pattern, " ", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected request pattern %q", r.Pattern)
	}
	method, pathPat := parts[0], parts[1]

	// Strip the provider base URL prefix.
	rest := strings.TrimPrefix(pathPat, baseURL)

	// Strip the tenant anchor: /v1/tenants/{...}
	rest = stripV1TenantsAnchor(rest)

	// Strip optional workspace anchor: /workspaces/{...}
	rest = stripWorkspaceAnchor(rest)

	// rest is now the resource portion of the pattern, e.g.:
	//   /instances
	//   /instances/{name}
	//   /instances/{name}/restart
	//   /networks/{network}/subnets/{name}
	segs := pathSegments(rest)

	// Derive verb and trim trailing name/action wildcards.
	switch method {
	case http.MethodPost:
		// POST action routes end with /{name}/<action> where action is a static segment.
		if len(segs) >= 2 && !isWildcard(segs[len(segs)-1]) && isWildcard(segs[len(segs)-2]) {
			action := segs[len(segs)-1]
			verb = "post." + action
			segs = segs[:len(segs)-2] // drop {name} and action
		} else {
			verb = "post"
			if len(segs) > 0 && isWildcard(segs[len(segs)-1]) {
				segs = segs[:len(segs)-1]
			}
		}

	case http.MethodGet:
		if name == "" {
			verb = "list"
			// Last segment is the collection name; keep it.
		} else {
			verb = "get"
			if len(segs) > 0 && isWildcard(segs[len(segs)-1]) {
				segs = segs[:len(segs)-1] // drop {name}
			}
		}

	case http.MethodPut:
		verb = "put"
		if len(segs) > 0 && isWildcard(segs[len(segs)-1]) {
			segs = segs[:len(segs)-1]
		}

	case http.MethodDelete:
		verb = "delete"
		if len(segs) > 0 && isWildcard(segs[len(segs)-1]) {
			segs = segs[:len(segs)-1]
		}

	default:
		verb = strings.ToLower(method)
		if len(segs) > 0 && isWildcard(segs[len(segs)-1]) {
			segs = segs[:len(segs)-1]
		}
	}

	// Collect only the static (non-wildcard) segments — these are the resource kind names.
	// Wildcards in the middle (e.g. {network} in "networks/{network}/subnets") are parent
	// resource references and are skipped.
	var kinds []string
	for _, s := range segs {
		if !isWildcard(s) {
			kinds = append(kinds, s)
		}
	}
	resource = strings.Join(kinds, "/")
	return resource, verb, nil
}

// isWildcard reports whether a path segment is a Go 1.22 mux wildcard (e.g. {name}).
func isWildcard(s string) bool {
	return len(s) > 2 && s[0] == '{' && s[len(s)-1] == '}'
}

// pathSegments splits a slash-separated path into non-empty segments.
func pathSegments(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

// stripV1TenantsAnchor removes the "/v1/tenants/{...}" prefix from a pattern path.
// Returns the remainder starting with "/" or "" when nothing follows the tenant wildcard.
func stripV1TenantsAnchor(path string) string {
	const prefix = "/v1/tenants/"
	if !strings.HasPrefix(path, prefix) {
		return path
	}
	rest := path[len(prefix):]
	i := strings.IndexByte(rest, '/')
	if i < 0 {
		return ""
	}
	return rest[i:] // "/workspaces/..." or "/resources/..."
}

// stripWorkspaceAnchor removes the "/workspaces/{...}" prefix when present.
// Returns the remainder starting with "/" or "" when nothing follows the workspace wildcard.
func stripWorkspaceAnchor(path string) string {
	const prefix = "/workspaces/"
	if !strings.HasPrefix(path, prefix) {
		return path
	}
	rest := path[len(prefix):]
	i := strings.IndexByte(rest, '/')
	if i < 0 {
		return ""
	}
	return rest[i:]
}
