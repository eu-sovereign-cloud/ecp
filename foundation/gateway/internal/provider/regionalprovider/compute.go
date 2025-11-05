package regionalprovider

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

var _ ComputeProvider = (*ComputeServer)(nil) // Ensure ComputeServer implements the ComputeProvider interface.

// ComputeServer implements the compute.ComputeProvider interface and provides methods to manage compute resources.
type ComputeServer struct{}

func (c ComputeServer) ListSkus(ctx context.Context, tid secapi.TenantID) (*secapi.Iterator[schema.InstanceSku], error) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeServer) GetSku(ctx context.Context, tref secapi.TenantReference) (*schema.InstanceSku, error) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeServer) ListInstances(ctx context.Context, tid secapi.TenantID, wid secapi.WorkspaceID) (*secapi.Iterator[schema.Instance], error) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeServer) GetInstance(ctx context.Context, wref secapi.WorkspaceReference) (*schema.Instance, error) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeServer) CreateOrUpdateInstance(ctx context.Context, inst *schema.Instance) (*schema.Instance, error) {
	// TODO implement me
	panic("implement me")
}

func (c ComputeServer) DeleteInstance(ctx context.Context, inst *schema.Instance) error {
	// TODO implement me
	panic("implement me")
}

func (c ComputeServer) StartInstance(ctx context.Context, inst *schema.Instance) error {
	// TODO implement me
	panic("implement me")
}

func (c ComputeServer) StopInstance(ctx context.Context, inst *schema.Instance) error {
	// TODO implement me
	panic("implement me")
}

func (c ComputeServer) RestartInstance(ctx context.Context, inst *schema.Instance) error {
	// TODO implement me
	panic("implement me")
}
