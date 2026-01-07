package repository

import (
	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StorageRepository = *CommonRepository[*v1alpha1.BlockStorage]

func NewStorageRepository(client client.Client) StorageRepository {
	return NewCommonRepository[*v1alpha1.BlockStorage](client)
}
