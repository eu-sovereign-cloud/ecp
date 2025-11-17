package port

import (
	"context"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/eu-sovereign-cloud/go-sdk/secapi"
)

type ComputeProvider interface {
	SKUProvider

	ListInstances(ctx context.Context, tenantID secapi.TenantID, workspaceID secapi.WorkspaceID) (*secapi.Iterator[schema.Instance], error)
	GetInstance(ctx context.Context, workspaceRef secapi.WorkspaceReference) (*schema.Instance, error)
	CreateOrUpdateInstance(ctx context.Context, inst *schema.Instance) (*schema.Instance, error)
	DeleteInstance(ctx context.Context, instance *schema.Instance) error
	StartInstance(ctx context.Context, instance *schema.Instance) error
	StopInstance(ctx context.Context, instance *schema.Instance) error
	RestartInstance(ctx context.Context, instance *schema.Instance) error
}

type SKUProvider interface {
	ListSkus(ctx context.Context, tenantID secapi.TenantID) (*secapi.Iterator[schema.InstanceSku], error)
	GetSku(ctx context.Context, tenantRef secapi.TenantReference) (*schema.InstanceSku, error)
}
