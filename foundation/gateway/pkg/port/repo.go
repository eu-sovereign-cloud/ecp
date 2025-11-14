package port

import (
	"context"
)

// ResourceIdentifier defines the interface for objects that can be identified
// by name and namespace
type ResourceIdentifier interface {
	GetName() string
	GetNamespace() string
	SetName(name string)
	SetNamespace(namespace string)
}

type ListParams struct {
	Namespace string
	Limit     int
	SkipToken string
	Selector  string
}

type Repo[T ResourceIdentifier] interface {
	Reader[T]
	Writer[T]
	Watcher[T]
}

type Writer[T ResourceIdentifier] interface {
	Delete(ctx context.Context, m T) error // model.ErrNotfound
	Create(ctx context.Context, m T) error // model.ErrConflict
	Update(ctx context.Context, m T) error // model.ErrNotfound
}

type Watcher[T any] interface {
	Watch(ctx context.Context, m chan<- T) error
}

type Reader[T ResourceIdentifier] interface {
	List(ctx context.Context, params ListParams, list *[]T) (*string, error)
	Load(ctx context.Context, m *T) error // model.ErrNotfound
}

type ResourceQueryRepository[T ResourceIdentifier] interface {
	Reader[T]
}
