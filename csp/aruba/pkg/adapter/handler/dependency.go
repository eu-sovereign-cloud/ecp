package handler

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	backend "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/repository"
)

// Dependency gates shared by the resource handlers. Every Aruba resource hangs off a Project,
// which the workspace handler creates concurrently, so "not there yet" is the normal case rather
// than a failure: these helpers report backend.ErrStillProcessing, which the reconciler turns
// into a requeue. Note that requeues carry no message, so the SECA resource stays in "creating"
// with no indication of which dependency is missing - see csp/aruba/README.md.

// loadActiveWorkspace loads the SECA Workspace owning scope and reports it only once active.
func loadActiveWorkspace(ctx context.Context, repo persistence.ReaderRepo[*wsdom.Workspace], scope persistence.Scope) (*wsdom.Workspace, error) {
	ws := &wsdom.Workspace{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{
				Name: scope.GetWorkspace(),
			},
			Scope: res.Scope{
				Tenant: scope.GetTenant(),
			},
		},
	}

	if err := repo.Load(ctx, &ws); err != nil {
		return nil, backend.ErrStillProcessing // TODO: better error handling
	}

	if ws.Status == nil || ws.Status.State != commondomain.ResourceStateActive {
		return nil, backend.ErrStillProcessing // TODO: better error handling
	}

	return ws, nil
}

// loadActiveProject loads the Aruba Project and reports whether it is usable as a parent.
func loadActiveProject(ctx context.Context, repo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList], prj *v1alpha1.Project) error {
	if err := repo.Load(ctx, prj); err != nil {
		if apierrors.IsNotFound(err) {
			return backend.ErrStillProcessing // Project not found, wait for it to be created
		}

		return err // Other errors should be returned for handling
	}

	if prj.Status.Phase != v1alpha1.ResourcePhaseActive {
		return backend.ErrStillProcessing // Project is not ready, wait for it to be active
	}

	return nil
}

// arubaResource is satisfied by every arubacloud.com CR: the operator embeds ResourceStatus in each
// of them and exposes it through this accessor.
type arubaResource interface {
	GetResourceStatus() *v1alpha1.ResourceStatus
}

// loadActiveAruba loads an arubacloud.com CR and reports it only once the operator has provisioned
// it. Aruba requires every resource a CloudServer references (boot and data volumes, key pair,
// security groups, elastic IP) to already exist in the CMP, so a server create issued while one is
// still pending is rejected outright rather than retried - gate on this instead.
func loadActiveAruba[T arubaResource, L any](ctx context.Context, repo repository.Repository[T, L], obj T) error {
	if err := repo.Load(ctx, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return backend.ErrStillProcessing // not created yet
		}

		return err
	}

	if obj.GetResourceStatus().Phase != v1alpha1.ResourcePhaseActive {
		return backend.ErrStillProcessing // still provisioning
	}

	return nil
}
