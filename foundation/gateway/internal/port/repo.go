package port

import (
	"context"

	model "github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/model"
)

type Repo[T any] interface {
	List(ctx context.Context, f *model.Filter) ([]T, error)
	GetByMetadataName(ctx context.Context, name string) (T, error)
	Delete(ctx context.Context, m T) error
	Create(ctx context.Context, m T) error
	Update(ctx context.Context, m T) error
}
