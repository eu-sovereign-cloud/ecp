package repository

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GenericRepository[T client.Object] struct {
	client client.Client
}

func NewGenericRepository[T client.Object](client client.Client) *GenericRepository[T] {
	return &GenericRepository[T]{client: client}
}

func (r *GenericRepository[T]) Create(ctx context.Context, resource T) error {
	return r.client.Create(ctx, resource)
}

func (r *GenericRepository[T]) Load(ctx context.Context, resource T) error {
	key, err := objectKeyFromResource(resource)
	if err != nil {
		return err
	}
	return r.client.Get(ctx, key, resource)
}

func (r *GenericRepository[T]) Update(ctx context.Context, resource T) error {
	return r.client.Update(ctx, resource)
}

func (r *GenericRepository[T]) Delete(ctx context.Context, resource T) error {
	return r.client.Delete(ctx, resource)
}

// ResolveReference loads a resource from a ResourceReference
func (r *GenericRepository[T]) ResolveReference(
	ctx context.Context,
	ref v1alpha1.ResourceReference,
	resource T) error {

	key := objectKeyFromResourceReference(ref)

	return r.client.Get(ctx, key, resource)
}

// Helpers
func objectKeyFromResource(resource client.Object) (client.ObjectKey, error) {
	meta, err := meta.Accessor(resource)
	if err != nil {
		return client.ObjectKey{}, err
	}
	return client.ObjectKey{
		Name:      meta.GetName(),
		Namespace: meta.GetNamespace(),
	}, nil
}

func objectKeyFromResourceReference(res v1alpha1.ResourceReference) client.ObjectKey {
	return client.ObjectKey{
		Name:      res.Name,
		Namespace: res.Namespace,
	}
}
