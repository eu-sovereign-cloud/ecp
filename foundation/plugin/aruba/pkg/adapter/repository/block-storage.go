package repository

import (
	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	generic_repository "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/repository"
)

type StorageRepository = *generic_repository.GenericRepository[*v1alpha1.BlockStorage]

func NewStorageRepository(client client.Client) StorageRepository {
	return generic_repository.NewGenericRepository[*v1alpha1.BlockStorage](client)
}
