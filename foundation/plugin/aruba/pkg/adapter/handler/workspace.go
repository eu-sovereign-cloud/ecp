package handler

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/delegated"
	mutator_bypass "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/mutator"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter"
	repository "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository"
)

// Ensure WorkspaceHandler implements the Workspace interface
var _ plugin.Workspace = (*WorkspaceHandler)(nil)

// WorkspaceHandler handles WorkspaceDomain resources by interacting with Aruba Projects.
// It is responsible for translating WorkspaceDomain resources to Aruba Projects
// and managing their lifecycle (Create/Delete).
type WorkspaceHandler struct {
	createDelegated *delegated.GenericDelegated[*regional.WorkspaceDomain, *regional.WorkspaceDomain, *v1alpha1.Project]
	deleteDelegated *delegated.GenericDelegated[*regional.WorkspaceDomain, *regional.WorkspaceDomain, *v1alpha1.Project]
}

// NewWorkspaceHandler creates a new WorkspaceHandler with the provided repository and converter.
// It sets up the necessary delegated operations for creating and deleting WorkspaceDomain resources.
// The handler uses bypass mutators since no mutation is needed on the Aruba Project objects.
func NewWorkspaceHandler(repo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList], conv converter.Converter[*regional.WorkspaceDomain, *v1alpha1.Project]) *WorkspaceHandler {
	return &WorkspaceHandler{
		createDelegated: delegated.NewStraightDelegated(
			conv.FromSECAToAruba,
			mutator_bypass.BypassMutateFunc[*v1alpha1.Project, *regional.WorkspaceDomain],
			func(ctx context.Context, ab *v1alpha1.Project) error {
				return repo.Create(ctx, ab)
			},
			func(p *v1alpha1.Project) bool {
				return p.Status.Phase == v1alpha1.ResourcePhaseCreated
			},
			repo.WaitUntil,
		),
		deleteDelegated: delegated.NewStraightDelegated(
			conv.FromSECAToAruba,
			mutator_bypass.BypassMutateFunc[*v1alpha1.Project, *regional.WorkspaceDomain],
			func(ctx context.Context, ab *v1alpha1.Project) error {
				return repo.Delete(ctx, ab)
			},
			func(p *v1alpha1.Project) bool {
				return p.Status.Phase == v1alpha1.ResourcePhaseDeleted
			},
			repo.WaitUntil,
		),
	}
}

// Create creates a new WorkspaceDomain by creating an Aruba Project.
func (h *WorkspaceHandler) Create(ctx context.Context, resource *regional.WorkspaceDomain) error {
	return h.createDelegated.Do(ctx, resource)
}

// Delete deletes an existing WorkspaceDomain by deleting the corresponding Aruba Project.
func (h *WorkspaceHandler) Delete(ctx context.Context, resource *regional.WorkspaceDomain) error {
	return h.deleteDelegated.Do(ctx, resource)
}
