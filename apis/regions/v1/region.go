package v1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Copied from here: https://github.com/eu-sovereign-cloud/go-sdk/blob/main/pkg/spec/foundation.region.v1/api.go
// todo: take automatically from the go-sdk
// Package region provides primitives to interact with the openapi HTTP API.

const (
	BearerAuthScopes = "bearerAuth.Scopes"
)

// Defines values for GlobalResourceMetadataKind.
const (
	GlobalResourceMetadataKindActivityLog          GlobalResourceMetadataKind = "activity-log"
	GlobalResourceMetadataKindBlockStorage         GlobalResourceMetadataKind = "block-storage"
	GlobalResourceMetadataKindImage                GlobalResourceMetadataKind = "image"
	GlobalResourceMetadataKindInstance             GlobalResourceMetadataKind = "instance"
	GlobalResourceMetadataKindInstanceSku          GlobalResourceMetadataKind = "instance-sku"
	GlobalResourceMetadataKindNetwork              GlobalResourceMetadataKind = "network"
	GlobalResourceMetadataKindNetworkLoadBalancer  GlobalResourceMetadataKind = "network-load-balancer"
	GlobalResourceMetadataKindNetworkSku           GlobalResourceMetadataKind = "network-sku"
	GlobalResourceMetadataKindNic                  GlobalResourceMetadataKind = "nic"
	GlobalResourceMetadataKindObjectStorageAccount GlobalResourceMetadataKind = "object-storage-account"
	GlobalResourceMetadataKindPublicIp             GlobalResourceMetadataKind = "public-ip"
	GlobalResourceMetadataKindRegion               GlobalResourceMetadataKind = "region"
	GlobalResourceMetadataKindRole                 GlobalResourceMetadataKind = "role"
	GlobalResourceMetadataKindRoleAssignment       GlobalResourceMetadataKind = "role-assignment"
	GlobalResourceMetadataKindRoutingTable         GlobalResourceMetadataKind = "routing-table"
	GlobalResourceMetadataKindSecurityGroup        GlobalResourceMetadataKind = "security-group"
	GlobalResourceMetadataKindSecurityGroupRule    GlobalResourceMetadataKind = "security-group-rule"
	GlobalResourceMetadataKindStorageSku           GlobalResourceMetadataKind = "storage-sku"
	GlobalResourceMetadataKindSubnet               GlobalResourceMetadataKind = "subnet"
	GlobalResourceMetadataKindWorkspace            GlobalResourceMetadataKind = "workspace"
)

// Defines values for TypeMetadataKind.
const (
	TypeMetadataKindActivityLog          TypeMetadataKind = "activity-log"
	TypeMetadataKindBlockStorage         TypeMetadataKind = "block-storage"
	TypeMetadataKindImage                TypeMetadataKind = "image"
	TypeMetadataKindInstance             TypeMetadataKind = "instance"
	TypeMetadataKindInstanceSku          TypeMetadataKind = "instance-sku"
	TypeMetadataKindNetwork              TypeMetadataKind = "network"
	TypeMetadataKindNetworkLoadBalancer  TypeMetadataKind = "network-load-balancer"
	TypeMetadataKindNetworkSku           TypeMetadataKind = "network-sku"
	TypeMetadataKindNic                  TypeMetadataKind = "nic"
	TypeMetadataKindObjectStorageAccount TypeMetadataKind = "object-storage-account"
	TypeMetadataKindPublicIp             TypeMetadataKind = "public-ip"
	TypeMetadataKindRegion               TypeMetadataKind = "region"
	TypeMetadataKindRole                 TypeMetadataKind = "role"
	TypeMetadataKindRoleAssignment       TypeMetadataKind = "role-assignment"
	TypeMetadataKindRoutingTable         TypeMetadataKind = "routing-table"
	TypeMetadataKindSecurityGroup        TypeMetadataKind = "security-group"
	TypeMetadataKindSecurityGroupRule    TypeMetadataKind = "security-group-rule"
	TypeMetadataKindStorageSku           TypeMetadataKind = "storage-sku"
	TypeMetadataKindSubnet               TypeMetadataKind = "subnet"
	TypeMetadataKindWorkspace            TypeMetadataKind = "workspace"
)

// Defines values for AcceptHeader.
const (
	AcceptHeaderApplicationjson            AcceptHeader = "application/json"
	AcceptHeaderApplicationjsonDeletedOnly AcceptHeader = "application/json; deleted=only"
	AcceptHeaderApplicationjsonDeletedTrue AcceptHeader = "application/json; deleted=true"
)

// Defines values for ListRegionsParamsAccept.
const (
	ListRegionsParamsAcceptApplicationjson            ListRegionsParamsAccept = "application/json"
	ListRegionsParamsAcceptApplicationjsonDeletedOnly ListRegionsParamsAccept = "application/json; deleted=only"
	ListRegionsParamsAcceptApplicationjsonDeletedTrue ListRegionsParamsAccept = "application/json; deleted=true"
)

// Error A detailed error response see https://datatracker.ietf.org/doc/html/rfc7807.
type Error struct {
	// Detail A human-readable explanation specific to this occurrence of the problem.
	Detail *string `json:"detail,omitempty"`

	// Instance A URI reference that identifies the specific occurrence of the problem.
	// It may or may not yield further information if dereferenced.
	Instance string `json:"instance"`

	// Meta A meta object containing non-standard meta-information about the error.
	Meta    *map[string]string `json:"meta,omitempty"`
	Sources *[]ErrorSource     `json:"sources,omitempty"`

	// Status The HTTP status type ([http://secapi.eu/errors/-rfc7231], Section 6)
	// generated by the origin server for this occurrence of the problem.
	Status float32 `json:"status"`

	// Title A short, human-readable summary of the problem
	// type.  It SHOULD NOT change from occurrence to occurrence of the
	// problem, except for purposes of localization (e.g., using
	// proactive content negotiation; see [RFC7231], Section 3.4).
	Title string `json:"title"`

	// Type The type of error, expressed as a URI.
	Type string `json:"type"`
}

// Error400 defines model for Error400.
type Error400 struct {
	// Detail A human-readable explanation specific to this occurrence of the problem.
	Detail *string `json:"detail,omitempty"`

	// Instance A URI reference that identifies the specific occurrence of the problem.
	// It may or may not yield further information if dereferenced.
	Instance string `json:"instance"`

	// Meta A meta object containing non-standard meta-information about the error.
	Meta    *map[string]string `json:"meta,omitempty"`
	Sources *[]ErrorSource     `json:"sources,omitempty"`

	// Status The HTTP status type ([http://secapi.eu/errors/-rfc7231], Section 6)
	// generated by the origin server for this occurrence of the problem.
	Status float32 `json:"status"`

	// Title A short, human-readable summary of the problem
	// type.  It SHOULD NOT change from occurrence to occurrence of the
	// problem, except for purposes of localization (e.g., using
	// proactive content negotiation; see [RFC7231], Section 3.4).
	Title string `json:"title"`

	// Type The type of error, expressed as a URI.
	Type string `json:"type"`
}

// Error401 defines model for Error401.
type Error401 struct {
	// Detail A human-readable explanation specific to this occurrence of the problem.
	Detail *string `json:"detail,omitempty"`

	// Instance A URI reference that identifies the specific occurrence of the problem.
	// It may or may not yield further information if dereferenced.
	Instance string `json:"instance"`

	// Meta A meta object containing non-standard meta-information about the error.
	Meta    *map[string]string `json:"meta,omitempty"`
	Sources *[]ErrorSource     `json:"sources,omitempty"`

	// Status The HTTP status type ([http://secapi.eu/errors/-rfc7231], Section 6)
	// generated by the origin server for this occurrence of the problem.
	Status float32 `json:"status"`

	// Title A short, human-readable summary of the problem
	// type.  It SHOULD NOT change from occurrence to occurrence of the
	// problem, except for purposes of localization (e.g., using
	// proactive content negotiation; see [RFC7231], Section 3.4).
	Title string `json:"title"`

	// Type The type of error, expressed as a URI.
	Type string `json:"type"`
}

// Error403 defines model for Error403.
type Error403 struct {
	// Detail A human-readable explanation specific to this occurrence of the problem.
	Detail *string `json:"detail,omitempty"`

	// Instance A URI reference that identifies the specific occurrence of the problem.
	// It may or may not yield further information if dereferenced.
	Instance string `json:"instance"`

	// Meta A meta object containing non-standard meta-information about the error.
	Meta    *map[string]string `json:"meta,omitempty"`
	Sources *[]ErrorSource     `json:"sources,omitempty"`

	// Status The HTTP status type ([http://secapi.eu/errors/-rfc7231], Section 6)
	// generated by the origin server for this occurrence of the problem.
	Status float32 `json:"status"`

	// Title A short, human-readable summary of the problem
	// type.  It SHOULD NOT change from occurrence to occurrence of the
	// problem, except for purposes of localization (e.g., using
	// proactive content negotiation; see [RFC7231], Section 3.4).
	Title string `json:"title"`

	// Type The type of error, expressed as a URI.
	Type string `json:"type"`
}

// Error404 defines model for Error404.
type Error404 struct {
	// Detail A human-readable explanation specific to this occurrence of the problem.
	Detail *string `json:"detail,omitempty"`

	// Instance A URI reference that identifies the specific occurrence of the problem.
	// It may or may not yield further information if dereferenced.
	Instance string `json:"instance"`

	// Meta A meta object containing non-standard meta-information about the error.
	Meta    *map[string]string `json:"meta,omitempty"`
	Sources *[]ErrorSource     `json:"sources,omitempty"`

	// Status The HTTP status type ([http://secapi.eu/errors/-rfc7231], Section 6)
	// generated by the origin server for this occurrence of the problem.
	Status float32 `json:"status"`

	// Title A short, human-readable summary of the problem
	// type.  It SHOULD NOT change from occurrence to occurrence of the
	// problem, except for purposes of localization (e.g., using
	// proactive content negotiation; see [RFC7231], Section 3.4).
	Title string `json:"title"`

	// Type The type of error, expressed as a URI.
	Type string `json:"type"`
}

// Error500 defines model for Error500.
type Error500 struct {
	// Detail A human-readable explanation specific to this occurrence of the problem.
	Detail *string `json:"detail,omitempty"`

	// Instance A URI reference that identifies the specific occurrence of the problem.
	// It may or may not yield further information if dereferenced.
	Instance string `json:"instance"`

	// Meta A meta object containing non-standard meta-information about the error.
	Meta    *map[string]string `json:"meta,omitempty"`
	Sources *[]ErrorSource     `json:"sources,omitempty"`

	// Status The HTTP status type ([http://secapi.eu/errors/-rfc7231], Section 6)
	// generated by the origin server for this occurrence of the problem.
	Status float32 `json:"status"`

	// Title A short, human-readable summary of the problem
	// type.  It SHOULD NOT change from occurrence to occurrence of the
	// problem, except for purposes of localization (e.g., using
	// proactive content negotiation; see [RFC7231], Section 3.4).
	Title string `json:"title"`

	// Type The type of error, expressed as a URI.
	Type string `json:"type"`
}

// ErrorSource An object containing references to the source of the error.
type ErrorSource struct {
	// Parameter A string indicating which URI query parameter caused the error.
	Parameter string `json:"parameter"`

	// Pointer A JSON Pointer [RFC6901] to the associated entity in the request document.
	Pointer string `json:"pointer"`
}

// GlobalResourceMetadata defines model for GlobalResourceMetadata.
type GlobalResourceMetadata struct {
	// ApiVersion API version of the resource
	ApiVersion string `json:"apiVersion"`

	// CreatedAt Indicates the time when the resource was created. The field is set by the provider and should not be modified by the user.
	CreatedAt metav1.Time `json:"createdAt"`

	// DeletedAt If set, indicates the time when the resource was marked for deletion. Resources with this field set are considered pending deletion.
	DeletedAt *metav1.Time `json:"deletedAt,omitempty"`

	// Kind Type of the resource
	Kind GlobalResourceMetadataKind `json:"kind"`

	// LastModifiedAt Indicates the time when the resource was created or last modified. Field is used for "If-Unmodified-Since" logic for concurrency control. The provider guarantees that a modification on a single resource can happen only once every millisecond.
	LastModifiedAt metav1.Time `json:"lastModifiedAt"`

	// Name Resource identifier in dash-case (kebab-case) format. Must start and end with an alphanumeric character.
	// Can contain lowercase letters, numbers, and hyphens. Multiple segments can be joined with dots.
	// Each segment follows the same rules.
	Name     string `json:"name"`
	Provider string `json:"provider"`

	// Ref Reference to a resource. The reference is represented as the full URN (Uniform Resource Name) name of the resource.
	// The reference can be used to refer to a resource in other resources.
	Ref      *Reference `json:"ref,omitempty"`
	Resource string     `json:"resource"`

	// ResourceVersion Incremented on every modification of the resource. Used for optimistic concurrency control.
	ResourceVersion int `json:"resourceVersion"`

	// Tenant Tenant identifier
	Tenant string `json:"tenant"`
	Verb   string `json:"verb"`
}

// GlobalResourceMetadataKind Type of the resource
type GlobalResourceMetadataKind string

// ModificationMetadata Base metadata for all resources with optional region references
type ModificationMetadata struct {
	// CreatedAt Indicates the time when the resource was created. The field is set by the provider and should not be modified by the user.
	CreatedAt metav1.Time `json:"createdAt"`

	// DeletedAt If set, indicates the time when the resource was marked for deletion. Resources with this field set are considered pending deletion.
	DeletedAt *metav1.Time `json:"deletedAt,omitempty"`

	// LastModifiedAt Indicates the time when the resource was created or last modified. Field is used for "If-Unmodified-Since" logic for concurrency control. The provider guarantees that a modification on a single resource can happen only once every millisecond.
	LastModifiedAt metav1.Time `json:"lastModifiedAt"`

	// ResourceVersion Incremented on every modification of the resource. Used for optimistic concurrency control.
	ResourceVersion int `json:"resourceVersion"`
}

// NameMetadata Metadata for resource names
type NameMetadata struct {
	// Name Resource identifier in dash-case (kebab-case) format. Must start and end with an alphanumeric character.
	// Can contain lowercase letters, numbers, and hyphens. Multiple segments can be joined with dots.
	// Each segment follows the same rules.
	Name string `json:"name"`
}

// PermissionMetadata Metadata for permission management
type PermissionMetadata struct {
	Provider string `json:"provider"`
	Resource string `json:"resource"`
	Verb     string `json:"verb"`
}

// Provider A provider of cloud services
type Provider struct {
	Name    string `json:"name"`
	Url     string `json:"url"`
	Version string `json:"version"`
}

// Reference Reference to a resource. The reference is represented as the full URN (Uniform Resource Resource) name of the resource.
// The reference can be used to refer to a resource in other resources.
type Reference struct {
	union json.RawMessage
}

// ReferenceObject A reference to a resource using an object. The object contains the
// same information as the ReferenceURN, but is represented as a structured object.
// The advantage of this representation is that it can be used to reference
// resources in different workspaces or regions without the need to specify
// the full URN.
type ReferenceObject struct {
	// Provider Provider of the resource. If not set, the provider is inferred from the context.
	Provider *string `json:"provider,omitempty"`

	// Region Region of the resource. If not set, the region is inferred from the context.
	Region *string `json:"region,omitempty"`

	// Resource Resource and type of the resource. Must be in the format `<type>/<name>`.
	// The type is the resource type, and the name is the resource name.
	Resource string `json:"resource"`

	// Tenant Tenant of the resource. If not set, the tenant is inferred from the context.
	Tenant *string `json:"tenant,omitempty"`

	// Workspace Workspace of the resource. If not set, the workspace is inferred from the context.
	Workspace *string `json:"workspace,omitempty"`
}

// ReferenceURN A unique resource name used to reference this resource in other resources. The reference
// is represented as the full URN (Uniform Resource Resource) name of the resource.
//
// ### Automatic Prefix Inference
//
// In most cases, the prefix of the URN can be automatically derived in the given context.
// To simplify usage, only the resource type and name might be specified as a reference
// using the `<type>/<name>` notation. The suffix can be made more specific by adding
// additional segments separated by slashes.
//
// The prefix is automatically inferred from the context. For example, if the resource is a
// block storage in the same workspace the reference can be specified as
// `block-storages/my-block-storage`. If the resource is a block storage in a different workspace, the
// reference can be specified as `workspaces/ws-1/block-storages/my-block-storage`.
//
// For automatic prefix inference, the following rules apply:
// - the version is inferred from the current resource version
// - the workspace is inferred from the current workspace
// - the region is inferred from the current region
// - the provider is inferred from the type and context of the usage
//
// The prefix inference is resolved on admission into the full URN format, which makes it
// mostly suitable for human use.
type ReferenceURN = string

// Region Represents a region, which is a geographical location
// with one or more zones.
type Region struct {
	// Metadata Metadata for global resources with name, permission, modification, type, and tenant information.
	Metadata *GlobalResourceMetadata `json:"metadata,omitempty"`

	// Spec The specification of a region, including the available zones and providers.
	Spec RegionSpec `json:"spec"`
}

// RegionIterator Iterator for regions
type RegionIterator struct {
	// Items List of regions
	Items []Region `json:"items"`

	// Metadata Metadata for response objects.
	Metadata ResponseMetadata `json:"metadata"`
}

// RegionSpec The specification of a region, including the available zones and providers.
type RegionSpec struct {
	// AvailableZones The list of zones available in the region.
	AvailableZones []Zone `json:"availableZones"`

	// Providers The list of providers available in the region.
	Providers []Provider `json:"providers"`
}

// ResponseMetadata defines model for ResponseMetadata.
type ResponseMetadata struct {
	Provider string `json:"provider"`
	Resource string `json:"resource"`

	// SkipToken Opaque cursor to get the next page. Field is omitted when there are no more pages available.
	SkipToken *string `json:"skipToken,omitempty"`
	Verb      string  `json:"verb"`
}

// TenantMetadata Metadata for global resources with tenant constraints
type TenantMetadata struct {
	// Tenant Tenant identifier
	Tenant string `json:"tenant"`
}

// TypeMetadata Metadata for all resources with type information.
type TypeMetadata struct {
	// ApiVersion API version of the resource
	ApiVersion string `json:"apiVersion"`

	// Kind Type of the resource
	Kind TypeMetadataKind `json:"kind"`

	// Ref Reference to a resource. The reference is represented as the full URN (Uniform Resource Resource) name of the resource.
	// The reference can be used to refer to a resource in other resources.
	Ref *Reference `json:"ref,omitempty"`
}

// TypeMetadataKind Type of the resource
type TypeMetadataKind string

// Zone Reference to a specific zone within a region
type Zone = string

// AcceptHeader defines model for acceptHeader.
type AcceptHeader string

// LabelSelector defines model for labelSelector.
type LabelSelector = string

// LimitParam defines model for limitParam.
type LimitParam = int

// ResourceName defines model for resourceName.
type ResourceName = string

// SkipTokenParam defines model for skipTokenParam.
type SkipTokenParam = string

// ListRegionsParams defines parameters for ListRegions.
type ListRegionsParams struct {
	// Labels Filter resources by their labels. Multiple filters are combined with comma.
	// Filter syntax:
	//   - Equals: key=value
	//   - Not equals: key!=value
	//   - Wildcards: *key*=*value* - matches if at least one pair match
	//   - Numeric: key>value, key<value, key>=value, key<=value
	//   - Namespaced key examples: 'monitoring:alert-level=high' or 'billing:team=platform'
	Labels *LabelSelector `form:"labels,omitempty" json:"labels,omitempty"`

	// Limit Maximum number of resources to return in the response
	Limit *LimitParam `form:"limit,omitempty" json:"limit,omitempty"`

	// SkipToken Opaque cursor for pagination. Use the skipToken from the previous response to get the next page of results. Note that skipTokens do not guarantee consistency across pages if the underlying data changes between requests
	SkipToken *SkipTokenParam `form:"skipToken,omitempty" json:"skipToken,omitempty"`

	// Accept Controls whether deleted resources are included:
	// - `"application/json"`: Returns only non-deleted resources
	// - `"application/json; deleted=true"`: Returns both deleted and non-deleted resources
	// - `"application/json; deleted=only"`: Returns only deleted resources
	Accept *ListRegionsParamsAccept `json:"Accept,omitempty"`
}

// ListRegionsParamsAccept defines parameters for ListRegions.
type ListRegionsParamsAccept string
