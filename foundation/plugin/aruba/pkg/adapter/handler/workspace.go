package handler

import (
	"context"
	"fmt"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter"
	repository "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository"
)

// Ensure WorkspaceHandler implements the Workspace interface
var _ plugin.Workspace = (*WorkspaceHandler)(nil)

// WorkspaceHandler handles WorkspaceDomain resources by interacting with Aruba Projects.
// It is responsible for translating WorkspaceDomain resources to Aruba Projects
// and managing their lifecycle (Create/Delete).
type WorkspaceHandler struct {
	repo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList]
	conv converter.Converter[*regional.WorkspaceDomain, *v1alpha1.Project]
}

// Create creates a new WorkspaceDomain by creating an Aruba Project.
func (h *WorkspaceHandler) Create(ctx context.Context, resource *regional.WorkspaceDomain) error {
	// Step 1: Convert WorkspaceDomain (gateway model) to Aruba Project (plugin model)
	project, err := h.conv.FromSECAToAruba(resource)
	if err != nil {
		return fmt.Errorf("failed to convert WorkspaceDomain to Project: %w", err)
	}

	// Step 2: Create the Project in the repository
	if err := h.repo.Create(ctx, project); err != nil {
		return fmt.Errorf("failed to create Project: %w", err)
	}

	// Step 3: Wait until the Project reaches the Created phase
	_, err = h.repo.WaitUntil(ctx, project, func(p *v1alpha1.Project) bool {
		//  Check if the project has reached the Created phase
		return p.Status.Phase == v1alpha1.ResourcePhaseCreated
	})

	// Step 4: Return any error encountered during waiting
	return err
}

// Delete deletes an existing WorkspaceDomain by deleting the corresponding Aruba Project.
func (h *WorkspaceHandler) Delete(ctx context.Context, resource *regional.WorkspaceDomain) error {
	// Step 1: Convert WorkspaceDomain (gateway model) to Aruba Project (plugin model)
	prj, err := h.conv.FromSECAToAruba(resource)

	if err != nil {
		return fmt.Errorf("failed to convert WorkspaceDomain to Project: %w", err)
	}

	// Step 2: Delete the Project in the repository
	if err := h.repo.Delete(ctx, prj); err != nil {
		return fmt.Errorf("failed to delete Project: %w", err)
	}

	// Step 3: Wait until the Project reaches the Deleted phase
	_, err = h.repo.WaitUntil(ctx, prj, func(p *v1alpha1.Project) bool {
		// Check if the project has reached the Deleted phase
		return p.Status.Phase == v1alpha1.ResourcePhaseDeleted
	})

	// Step 4: Return any error encountered during waiting
	return err
}
