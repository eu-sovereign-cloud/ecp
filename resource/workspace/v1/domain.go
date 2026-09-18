// Package v1 defines the workspace resource domain model and identity constants.
package v1

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

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

// GetRegion makes the workspace's region part of the key its backend stores it under, so one
// tenant can hold a workspace of the same name in two regions of a multi-region deployment.
//
// It is deliberately on Workspace and not on the embedded domain.RegionalMetadata: every other
// regional resource is keyed by tenant/workspace alone and would change namespace if it grew
// this method. See doc/ARCHITECTURE.md#multi-region-gateways.
func (w *Workspace) GetRegion() string { return w.Region }

// WorkspaceSpec is the free-form spec for a workspace.
type WorkspaceSpec = map[string]any

// WorkspaceStatus defines the status for a workspace.
type WorkspaceStatus struct {
	domain.Status
	ResourceCount *int
}
