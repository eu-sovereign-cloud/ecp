package repository

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CommonRepository[T client.Object] struct {
	client client.Client
}

func NewCommonRepository[T client.Object](client client.Client) *CommonRepository[T] {
	return &CommonRepository[T]{client: client}
}

func (r *CommonRepository[T]) Create(ctx context.Context, resource T) error {
	return r.client.Create(ctx, resource)
}

func (r *CommonRepository[T]) Load(ctx context.Context, resource T) error {

	objectKey := client.ObjectKeyFromObject(resource)

	return r.client.Get(ctx, objectKey, resource)
}

func (r *CommonRepository[T]) Update(ctx context.Context, resource T) error {
	return r.client.Update(ctx, resource)
}

func (r *CommonRepository[T]) Delete(ctx context.Context, resource T) error {
	return r.client.Delete(ctx, resource)
}
