package repository

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/repository"
)

type VPCRepository = *repository.GenericRepository[*v1alpha1.VPC, *v1alpha1.VPCList]

func NewVPCRepository(ctx context.Context, cli client.Client, ca crcache.Cache) VPCRepository {
	return repository.NewGenericRepository[*v1alpha1.VPC, *v1alpha1.VPCList](ctx, cli, ca)
}
