package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository"

	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GenericRepository is a generic repository for Kubernetes resources.
// It supports basic CRUD operations, listing, watching, and waiting for conditions.
// T represents the resource type (e.g., Deployment), L represents the resource list type (e.g., DeploymentList).
type GenericRepository[T client.Object, L client.ObjectList] struct {
	client client.Client
	list   L
}

// NewGenericRepository creates a new instance of GenericRepository.
func NewGenericRepository[T client.Object, L client.ObjectList](client client.Client, list L) *GenericRepository[T, L] {
	return &GenericRepository[T, L]{client: client, list: list}
}

// Create adds a new resource of type T to the cluster.
func (r *GenericRepository[T, L]) Create(ctx context.Context, resource T) error {
	return r.client.Create(ctx, resource)
}

// Load retrieves a resource of type T from the cluster based on its name and namespace.
func (r *GenericRepository[T, L]) Load(ctx context.Context, resource T) error {
	key, err := objectKeyFromResource(resource)
	if err != nil {
		return err
	}
	return r.client.Get(ctx, key, resource)
}

// List retrieves all resources of type T from the cluster.
// Accepts optional client.ListOption filters (e.g., namespace, labels).
func (r *GenericRepository[T, L]) List(ctx context.Context, opts ...client.ListOption) ([]T, error) {
	// var list L // L is already a pointer type like *ProjectList
	// list = reflect.New(reflect.TypeOf(list).Elem()).Interface().(L)

	if err := r.client.List(ctx, r.list, opts...); err != nil {
		return nil, err
	}

	items, err := extractItems[T](r.list)
	if err != nil {
		return nil, err
	}

	return items, nil

}

// Update modifies the specified resource in the cluster.
func (r *GenericRepository[T, L]) Update(ctx context.Context, resource T) error {
	return r.client.Update(ctx, resource)
}

// Delete removes the specified resource from the cluster.
func (r *GenericRepository[T, L]) Delete(ctx context.Context, resource T) error {
	return r.client.Delete(ctx, resource)
}

// WaitUntil polls the resource until the provided condition function returns true.
// It returns the latest resource state when the condition is met or an error if the context expires.
func (r *GenericRepository[T, L]) WaitUntil(
	ctx context.Context,
	resource T,
	condition repository.WaitConditionFunc[T],
) (T, error) {

	out, cancel, err := r.Watch(ctx, resource)
	if err != nil {
		var zero T
		return zero, err
	}
	defer cancel()

	for {
		select {
		case res, ok := <-out:
			if !ok {
				var zero T
				return zero, fmt.Errorf("watch channel closed")
			}
			if condition(res) {
				return res, nil
			}
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		}
	}
}

// Watch sets up a polling-based watch on the specified resource.
// It returns a channel receiving updated resource instances and a cancel function to stop watching.
func (r *GenericRepository[T, L]) Watch(
	ctx context.Context,
	resource T,
) (chan T, func(), error) {
	out := make(chan T)
	ctx, cancel := context.WithCancel(ctx)

	// Start a goroutine for polling
	go func() {
		defer close(out)

		ticker := time.NewTicker(5 * time.Second) // Polling interval
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				newRes := resource.DeepCopyObject().(T)

				err := r.Load(ctx, newRes)
				if err != nil {
					// Skip this iteration if resource is not available
					continue
				}

				select {
				case out <- newRes:
				case <-ctx.Done():
					return
				default:
					// If the channel is full, skip sending to avoid blocking
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return out, cancel, nil
}

// ResolveReference loads a resource from a ResourceReference
func (r *GenericRepository[T, L]) ResolveReference(
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

func extractItems[T client.Object](list client.ObjectList) ([]T, error) {
	itemsPtr, err := meta.ExtractList(list)
	if err != nil {
		return nil, err
	}

	result := make([]T, 0, len(itemsPtr))
	for _, item := range itemsPtr {
		obj, ok := item.(T)
		if !ok {
			return nil, fmt.Errorf("unexpected list item type %T", item)
		}
		result = append(result, obj)
	}
	return result, nil
}
