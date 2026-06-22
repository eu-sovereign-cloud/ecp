// Package domain defines the workspace resource domain model and identity constants.
package domain

import "github.com/eu-sovereign-cloud/ecp/resources/common/domain"

// Identity constants for the workspace resource.
const (
	Kind       = "Workspace"
	Resource   = "workspaces"
	Group      = "workspace.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.workspace/v1"
)

// WorkspaceDomain represents the domain model for a workspace.
type WorkspaceDomain struct {
	domain.RegionalMetadata

	Spec   WorkspaceSpecDomain
	Status *WorkspaceStatusDomain
}

// WorkspaceSpecDomain is the free-form spec for a workspace.
type WorkspaceSpecDomain = map[string]interface{}

// WorkspaceStatusDomain defines the status for a workspace.
type WorkspaceStatusDomain struct {
	domain.StatusDomain
	ResourceCount *int
}
