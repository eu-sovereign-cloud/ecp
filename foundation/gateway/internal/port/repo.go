package port

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/model"
)

type Repo[T any] interface {
	List(ctx context.Context, f *model.Filter) ([]T, error)
	GetByMetadataName(ctx context.Context, name string) (T, error)
	Delete(ctx context.Context, m T) error
	Create(ctx context.Context, m T) error
	Update(ctx context.Context, m T) error
}

type ListParams struct {
	Namespace string
	Limit     int
	SkipToken string
	Selector  string
}

type ResourceQueryRepository[T any] interface {
	List(ctx context.Context, params ListParams) ([]T, *string, error)
	Get(ctx context.Context, namespace, name string) (T, error)
}
