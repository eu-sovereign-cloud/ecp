package repository

import (
	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	generic_repository "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/repository"
)

// ProjectRepository is a typed alias for CommonRepository specialized for Project.
type ProjectRepository = generic_repository.GenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList]

// NewProjectRepository creates a new instance of ProjectRepository.
func NewProjectRepository(c client.Client, cache generic_repository.Cache) *ProjectRepository {
	return generic_repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](c, cache)
}
