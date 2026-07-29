package resource

// Scope defines the scope of a resource with tenant and workspace.
// It implements the persistence port's Scope interface.
type Scope struct {
	Tenant    string
	Workspace string
}

func (r Scope) GetTenant() string    { return r.Tenant }
func (r Scope) GetWorkspace() string { return r.Workspace }

// TokenScope is the optional down-scoping cap a bearer token asserts about itself.
//
// It is distinct from Scope: Scope identifies the single tenant/workspace one resource
// belongs to (a scalar identity), whereas TokenScope is a cap over the request's scope —
// each dimension is a list of permitted values, and an empty (nil) list leaves that
// dimension unconstrained. It also covers Region, which Scope does not.
//
// A TokenScope is carried verbatim by the authenticated identity and copied into the
// authorization claim, where it can only narrow the permissions granted by RBAC — it never
// grants anything. Both the authn and authz ports reuse this single type so the token cap
// has one definition.
//
// The json tags describe the Dummy authenticator's token wire format,
// base64(JSON{…,"scope":{"tenants":[…],"regions":[…],"workspaces":[…]}}), so token payloads
// unmarshal directly into it without a separate DTO.
type TokenScope struct {
	// Tenants restricts the token to the listed tenants; empty means any tenant.
	Tenants []string `json:"tenants,omitempty"`
	// Regions restricts the token to the listed regions; empty means any region.
	Regions []string `json:"regions,omitempty"`
	// Workspaces restricts the token to the listed workspaces; empty means any workspace.
	Workspaces []string `json:"workspaces,omitempty"`
}
