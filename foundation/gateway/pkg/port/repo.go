package port

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// NamespacedResource defines the interface for objects that can be identified
// by name and namespace
type NamespacedResource interface {
	GetName() string
	GetNamespace() string
	SetName(name string)
	SetNamespace(namespace string)
}

type Repo[T NamespacedResource] interface {
	Reader[T]
	Writer[T]
	Watcher[T]
}

type Writer[T NamespacedResource] interface {
	Delete(ctx context.Context, m T) error
	Create(ctx context.Context, m T) error
	Update(ctx context.Context, m T) error
}

type Watcher[T any] interface {
	Watch(ctx context.Context, m chan<- T) error
}

type Reader[T NamespacedResource] interface {
	List(ctx context.Context, params model.ListParams, list *[]T) (*string, error)
	Load(ctx context.Context, m *T) error
}

type ResourceQueryRepository[T NamespacedResource] interface {
	Reader[T]
}
