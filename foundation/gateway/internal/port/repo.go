package port

import (
	"context"

	model "github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/model"
)

type Repo[T any] interface {
	Reader[T]
	Writer[T]
	Watcher[T]
}

type Reader[T any] interface {
	List(ctx context.Context, f *model.Filter) ([]T, error)
	Load(ctx context.Context, m T) error // model.ErrNotfound
}

type Writer[T any] interface {
	Delete(ctx context.Context, m T) error // model.ErrNotfound
	Create(ctx context.Context, m T) error // model.ErrConflict
	Update(ctx context.Context, m T) error // model.ErrNotfound
}

type Watcher[T any] interface {
	Watch(ctx context.Context, m chan<- T) error
}
