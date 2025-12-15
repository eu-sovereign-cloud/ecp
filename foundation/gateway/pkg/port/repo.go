package port

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// IdentifiableResource defines the interface for objects that can be identified by name, tenant, and workspace
type IdentifiableResource interface {
	GetName() string
	SetName(name string)
	Scope
}

// Scope defines the interface for scoping the search of resources within tenant and workspace contexts.
type Scope interface {
	GetTenant() string
	GetWorkspace() string
	SetTenant(tenant string)
	SetWorkspace(workspace string)
}

type Repo[T IdentifiableResource] interface {
	Reader[T]
	Writer[T]
	Watcher[T]
}

type Writer[T IdentifiableResource] interface {
	Delete(ctx context.Context, m T) error
	Create(ctx context.Context, m T) error
	Update(ctx context.Context, m T) error
}

type Watcher[T any] interface {
	Watch(ctx context.Context, m chan<- T) error
}

type Reader[T IdentifiableResource] interface {
	List(ctx context.Context, params model.ListParams, list *[]T) (*string, error)
	Load(ctx context.Context, m *T) error
}

type ResourceQueryRepository[T IdentifiableResource] interface {
	Reader[T]
}
