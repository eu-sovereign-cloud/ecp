// Package v1 defines the workspace resource domain model and identity constants.
package v1

import "github.com/eu-sovereign-cloud/ecp/resources/common/domain"

// Identity constants for the workspace resource.
const (
	Kind       = "Workspace"
	Resource   = "workspaces"
	Group      = "workspace.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.workspace/v1"
)

// Workspace represents the domain model for a workspace.
type Workspace struct {
	domain.RegionalMetadata

	Spec   WorkspaceSpec
	Status *WorkspaceStatus
}

// WorkspaceSpec is the free-form spec for a workspace.
type WorkspaceSpec = map[string]interface{}

// WorkspaceStatus defines the status for a workspace.
type WorkspaceStatus struct {
	domain.StatusDomain
	ResourceCount *int
}
