package persistence

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

// Scope defines the interface for scoping the search of resources within tenant and workspace contexts.
// A resource can belong to a tenant, a workspace within a tenant, or be global (no tenant/workspace).
// There can be no workspaces without a tenant.
type Scope interface {
	GetTenant() string
	GetWorkspace() string
}

// IdentifiableResource defines the interface for objects that can be identified by name, tenant, and workspace.
type IdentifiableResource interface {
	GetName() string
	GetVersion() string
	Scope
}

// Repo is the combined repository interface for a resource.
type Repo[T IdentifiableResource] interface {
	ReaderRepo[T]
	WriterRepo[T]
	WatcherRepo[T]
}

// WriterRepo is the write-side repository interface.
type WriterRepo[T IdentifiableResource] interface {
	Delete(ctx context.Context, m T) error
	Create(ctx context.Context, m T) (*T, error)
	Update(ctx context.Context, m T) (*T, error)
	UpdateStatus(ctx context.Context, m T) (*T, error)
}

// WatcherRepo is the watch-side repository interface.
type WatcherRepo[T any] interface {
	Watch(ctx context.Context, m chan<- T) error
}

// ReaderRepo is the read-side repository interface.
type ReaderRepo[T IdentifiableResource] interface {
	List(ctx context.Context, params resource.ListParams, list *[]T) (*string, error)
	Load(ctx context.Context, m *T) error
}
