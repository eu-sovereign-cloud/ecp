package handler

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"

	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/delegated"
	mutator_bypass "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/mutator"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/converter"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/repository"
)

// Ensure WorkspaceHandler implements the Workspace interface
var _ wsk8s.WorkspacePlugin = (*WorkspaceHandler)(nil)

// WorkspaceHandler handles Workspace resources by interacting with Aruba Projects.
// It is responsible for translating Workspace resources to Aruba Projects
// and managing their lifecycle (Create/Delete).
type WorkspaceHandler struct {
	repository      repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList]
	converter       converter.Converter[*wsdom.Workspace, *v1alpha1.Project]
	createDelegated *delegated.GenericDelegated[*wsdom.Workspace, *wsdom.Workspace, *v1alpha1.Project]
	deleteDelegated *delegated.GenericDelegated[*wsdom.Workspace, *wsdom.Workspace, *v1alpha1.Project]
}

// NewWorkspaceHandler creates a new WorkspaceHandler with the provided repository and converter.
// It sets up the necessary delegated operations for creating and deleting Workspace resources.
// The handler uses bypass mutators since no mutation is needed on the Aruba Project objects.
func NewWorkspaceHandler(repo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList], conv converter.Converter[*wsdom.Workspace, *v1alpha1.Project]) *WorkspaceHandler {
	handler := &WorkspaceHandler{
		repository: repo,
		converter:  conv,
	}
	handler.createDelegated = delegated.NewStraightDelegated(
		conv.FromSECAToAruba,
		mutator_bypass.BypassMutateFunc[*v1alpha1.Project, *wsdom.Workspace],
		handler.propagateCreate,
		handler.checkWsCreated,
	)
	handler.deleteDelegated = delegated.NewStraightDelegated(
		conv.FromSECAToAruba,
		mutator_bypass.BypassMutateFunc[*v1alpha1.Project, *wsdom.Workspace],
		handler.propagateDelete,
		handler.checkWsDeleted,
	)

	return handler
}

// Create creates a new Workspace by creating an Aruba Project.
func (h *WorkspaceHandler) Create(ctx context.Context, domain *wsdom.Workspace) error {
	return h.createDelegated.Do(ctx, domain)
}

// Delete deletes an existing Workspace by deleting the corresponding Aruba Project.
func (h *WorkspaceHandler) Delete(ctx context.Context, domain *wsdom.Workspace) error {
	return h.deleteDelegated.Do(ctx, domain)
}

// checkWsCreated reports whether the Aruba Project already exists and has
// reached the active phase.
func (h *WorkspaceHandler) checkWsCreated(ctx context.Context, _ *wsdom.Workspace, project *v1alpha1.Project) (bool, error) {
	observed := project.DeepCopy()

	if err := h.repository.Load(ctx, observed); err != nil {
		if errors.IsNotFound(err) {
			return false, nil // Not created yet, it must be created.
		}

		return false, err
	}

	return observed.Status.Phase == v1alpha1.ResourcePhaseActive, nil
}

// checkWsDeleted reports whether the Aruba Project is gone.
func (h *WorkspaceHandler) checkWsDeleted(ctx context.Context, _ *wsdom.Workspace, project *v1alpha1.Project) (bool, error) {
	observed := project.DeepCopy()

	if err := h.repository.Load(ctx, observed); err != nil {
		if errors.IsNotFound(err) {
			return true, nil // Gone, deletion is complete.
		}

		return false, err
	}

	return false, nil // Still present, deletion is in progress.
}

// propagateCreate creates the Aruba Project. It is idempotent: because the
// create is (re)issued on every pass until the project becomes active, an
// already existing project is not treated as an error.
func (h *WorkspaceHandler) propagateCreate(ctx context.Context, project *v1alpha1.Project) error {
	if err := h.repository.Create(ctx, project); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// propagateDelete deletes the Aruba Project. It is idempotent: because the
// delete is (re)issued on every pass until the project is gone, an already
// missing project is not treated as an error.
func (h *WorkspaceHandler) propagateDelete(ctx context.Context, project *v1alpha1.Project) error {
	if err := h.repository.Delete(ctx, project); err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}
