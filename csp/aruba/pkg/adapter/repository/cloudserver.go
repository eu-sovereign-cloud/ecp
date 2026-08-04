package repository

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/repository"
)

type CloudServerRepository = *repository.GenericRepository[*v1alpha1.CloudServer, *v1alpha1.CloudServerList]

func NewCloudServerRepository(ctx context.Context, cli client.Client, ca crcache.Cache) CloudServerRepository {
	return repository.NewGenericRepository[*v1alpha1.CloudServer, *v1alpha1.CloudServerList](ctx, cli, ca)
}
