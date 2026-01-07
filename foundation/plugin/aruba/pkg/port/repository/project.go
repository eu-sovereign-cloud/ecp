package repository

import (
	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProjectRepository is a typed alias for CommonRepository specialized for Project.
type ProjectRepository = CommonRepository[*v1alpha1.Project]

// NewProjectRepository creates a new instance of ProjectRepository.
func NewProjectRepository(c client.Client) *ProjectRepository {
	return NewCommonRepository[*v1alpha1.Project](c)
}
