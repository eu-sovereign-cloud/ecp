package handler

import (
	"context"
	"time"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"

	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
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
	repository      repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList]
	converter       converter.Converter[*regional.WorkspaceDomain, *v1alpha1.Project]
	createDelegated *delegated.GenericDelegated[*regional.WorkspaceDomain, *regional.WorkspaceDomain, *v1alpha1.Project]
	deleteDelegated *delegated.GenericDelegated[*regional.WorkspaceDomain, *regional.WorkspaceDomain, *v1alpha1.Project]
}

// NewWorkspaceHandler creates a new WorkspaceHandler with the provided repository and converter.
// It sets up the necessary delegated operations for creating and deleting WorkspaceDomain resources.
// The handler uses bypass mutators since no mutation is needed on the Aruba Project objects.
func NewWorkspaceHandler(repo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList], conv converter.Converter[*regional.WorkspaceDomain, *v1alpha1.Project]) *WorkspaceHandler {
	handler := &WorkspaceHandler{
		repository: repo,
		converter:  conv,
	}
	handler.createDelegated = delegated.NewStraightDelegated(
		conv.FromSECAToAruba,
		mutator_bypass.BypassMutateFunc[*v1alpha1.Project, *regional.WorkspaceDomain],
		repo.Create,
		func(p *v1alpha1.Project) bool {
			return p.Status.Phase == v1alpha1.ResourcePhaseCreated
		},
		handler.waitUntilManagedError,
	)
	handler.deleteDelegated = delegated.NewStraightDelegated(
		conv.FromSECAToAruba,
		mutator_bypass.BypassMutateFunc[*v1alpha1.Project, *regional.WorkspaceDomain],
		repo.Delete,
		handler.checkWsDeleteCondition,
		handler.waitUntilManagedError,
	)

	return handler
}

// Create creates a new WorkspaceDomain by creating an Aruba Project.
func (h *WorkspaceHandler) Create(ctx context.Context, resource *regional.WorkspaceDomain) error {
	return h.createDelegated.Do(ctx, resource)
}

// Delete deletes an existing WorkspaceDomain by deleting the corresponding Aruba Project.
func (h *WorkspaceHandler) Delete(ctx context.Context, resource *regional.WorkspaceDomain) error {
	return h.deleteDelegated.Do(ctx, resource)
}

func (h *WorkspaceHandler) checkWsDeleteCondition(resource *v1alpha1.Project) bool {
	//TODO: refactor design completely
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := h.repository.Load(ctx, resource)

	return errors.IsNotFound(err)
}

// waitUntilManagedError waits until the provided condition is met for the given resource.
// If the condition is not met within the timeout, it returns delegator.ErrStillProcessing to indicate that the operation is still in progress.
func (h *WorkspaceHandler) waitUntilManagedError(ctx context.Context, resource *v1alpha1.Project, condition repository.WaitConditionFunc[*v1alpha1.Project]) (*v1alpha1.Project, error) {

	proj, err := h.repository.WaitUntil(ctx, resource, condition)

	if err != nil {
		// Check if the error is due to the resource not being found, which can be expected during deletion
		if errors.IsTimeout(err) {
			return nil, delegator.ErrStillProcessing // Resource is gone, treat as successful deletion
		}
		return nil, err // Return other errors for handling
	}

	return proj, nil
}
