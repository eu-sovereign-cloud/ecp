// Package role defines the role resource domain model and identity constants.
package role

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the role resource.
const (
	Kind     = "Role"
	Resource = "roles"
	Group    = "authorization.v1.secapi.cloud"
	Version  = "v1"

	AuthorizationBaseURL = "/providers/seca.authorization"
	ProviderID           = "seca.authorization/v1"
)

// Permission represents a single access control permission.
type Permission struct {
	Provider  string
	Resources []string
	Verb      []string
}

// RoleSpec defines the specification for a role.
type RoleSpec struct {
	Permissions []Permission
}

// RoleStatus defines the status for a role.
type RoleStatus struct {
	domain.Status
}

// Role is the domain model for a role resource.
type Role struct {
	domain.GlobalTenantMetadata
	Spec   RoleSpec
	Status *RoleStatus
}
