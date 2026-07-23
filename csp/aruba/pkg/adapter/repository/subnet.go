package repository

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/repository"
)

type SubnetRepository = *repository.GenericRepository[*v1alpha1.Subnet, *v1alpha1.SubnetList]

func NewSubnetRepository(ctx context.Context, cli client.Client, ca crcache.Cache) SubnetRepository {
	return repository.NewGenericRepository[*v1alpha1.Subnet, *v1alpha1.SubnetList](ctx, cli, ca)
}
