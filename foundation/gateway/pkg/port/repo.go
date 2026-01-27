package port

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// IdentifiableResource defines the interface for objects that can be identified by name, tenant, and workspace
type IdentifiableResource interface {
	GetName() string
	GetVersion() string
	Scope
}

// Scope defines the interface for scoping the search of resources within tenant and workspace contexts.
// A resource can belong to a tenant, a workspace within a tenant, or be global (no tenant/workspace).
// There can be no workspaces without a tenant.
type Scope interface {
	GetTenant() string
	GetWorkspace() string
}

type Repo[T IdentifiableResource] interface {
	ReaderRepo[T]
	WriterRepo[T]
	WatcherRepo[T]
}

type WriterRepo[T IdentifiableResource] interface {
	Delete(ctx context.Context, m T) error
	Create(ctx context.Context, m T) (*T, error)
	Update(ctx context.Context, m T) (*T, error)
}

type WatcherRepo[T any] interface {
	Watch(ctx context.Context, m chan<- T) error
}

type ReaderRepo[T IdentifiableResource] interface {
	List(ctx context.Context, params model.ListParams, list *[]T) (*string, error)
	Load(ctx context.Context, m *T) error
}
