package repository

import (
	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	generic_repository "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/repository"
)

type StorageRepository = *generic_repository.GenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList]

func NewStorageRepository(client client.Client, cache crcache.Cache) StorageRepository {
	return generic_repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](client, cache)
}
