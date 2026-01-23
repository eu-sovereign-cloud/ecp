package repository

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/repository"
)

// ProjectRepository is a typed alias for CommonRepository specialized for Project.
type ProjectRepository = repository.GenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList]

// NewProjectRepository creates a new instance of ProjectRepository.
func NewProjectRepository(ctx context.Context, cli client.Client, ca crcache.Cache) *ProjectRepository {
	return repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctx, cli, ca)
}
