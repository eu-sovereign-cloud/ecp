// Package roleassignment defines the role assignment resource domain model and identity constants.
package roleassignment

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the role assignment resource.
const (
	Kind       = "RoleAssignment"
	Resource   = "role-assignments"
	Group      = "authorization.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.authorization/v1"
)

// RoleAssignment represents the domain model for a role assignment.
type RoleAssignment struct {
	domain.RegionalMetadata
	Spec   RoleAssignmentSpec
	Status *RoleAssignmentStatus
}

// RoleAssignmentSpec defines the specification for a role assignment.
type RoleAssignmentSpec struct {
	Subs   []string
	Scopes []RoleAssignmentScope
	Roles  []string
}

// RoleAssignmentScope defines a single scope (tenants, regions, workspaces) for a role assignment.
type RoleAssignmentScope struct {
	Tenants    []string
	Regions    []string
	Workspaces []string
}

// RoleAssignmentStatus defines the status for a role assignment.
type RoleAssignmentStatus struct {
	domain.Status
}
