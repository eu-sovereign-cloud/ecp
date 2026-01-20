package repository

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository"

	"k8s.io/apimachinery/pkg/api/meta"
	kcache "k8s.io/client-go/tools/cache"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GenericRepository is a generic repository for Kubernetes resources.
// It supports basic CRUD operations, listing, watching, and waiting for conditions.
// T represents the resource type (e.g., Deployment), L represents the resource list type (e.g., DeploymentList).
type GenericRepository[T client.Object, L client.ObjectList] struct {
	client client.Client
	cache  crcache.Cache
	mu     sync.Mutex
}

// NewGenericRepository creates a new instance of GenericRepository.
func NewGenericRepository[T client.Object, L client.ObjectList](client client.Client, cache crcache.Cache) *GenericRepository[T, L] {
	return &GenericRepository[T, L]{client: client, cache: cache}
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
func (r *GenericRepository[T, L]) List(ctx context.Context, opts ...client.ListOption) (L, error) {
	var list L
	list = reflect.New(reflect.TypeOf(list).Elem()).Interface().(L)
	// list := r.list()

	if err := r.client.List(ctx, list, opts...); err != nil {
		return list, err
	}

	return list, nil

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
	defer cancel()
	if err != nil {
		var zero T
		return zero, err
	}

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

	ictx, cancel := context.WithCancel(ctx)

	// Create a zero value of T to get the informer
	objType := reflect.New(reflect.TypeOf(resource).Elem()).Interface().(T)

	informer, err := r.cache.GetInformer(ctx, objType)
	if err != nil {
		fmt.Printf("error getting informer: %v\n", err)
		return nil, cancel, err
	}

	handler := &kcache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			r.handle(ictx, obj, resource, out)
		},
		UpdateFunc: func(_, newObj interface{}) {
			r.handle(ictx, newObj, resource, out)
		},
		DeleteFunc: func(obj interface{}) {
			r.handle(ictx, obj, resource, out)
		},
	}

	r.mu.Lock()
	r.addEventHandler(informer, handler)
	r.mu.Unlock()

	go func() {
		<-ictx.Done()
		close(out)
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

// addEventHandler adds the event handler to the informer
func (r *GenericRepository[T, L]) addEventHandler(
	informer crcache.Informer,
	handler kcache.ResourceEventHandler,
) {
	informer.AddEventHandler(handler)
}

// handle processes an event object and sends it to the output channel if it matches the filter
func (r *GenericRepository[T, L]) handle(ctx context.Context,
	obj interface{},
	filter T,
	out chan T,
) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Type assert the object to T
	co, ok := obj.(T)
	if !ok {
		return
	}
	// Check if the object matches the filter
	if !matches(co, filter) {
		return
	}

	// Deep copy the object before sending
	copied, ok := co.DeepCopyObject().(T)
	if !ok {
		return
	}

	// Send the copied object to the output channel
	select {
	case <-ctx.Done():
		return
	case out <- copied:
	default:
		// drop event if consumer is slow
	}
}

// matches checks if the obj matches the filter based on Name and Namespace
func matches(obj, filter client.Object) bool {
	if filter.GetName() != "" && obj.GetName() != filter.GetName() {
		return false
	}
	if filter.GetNamespace() != "" && obj.GetNamespace() != filter.GetNamespace() {
		return false
	}
	return true
}

// Helpers
// objectKeyFromResource creates a client.ObjectKey from a client.Object
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

// objectKeyFromResourceReference creates a client.ObjectKey from a ResourceReference
func objectKeyFromResourceReference(res v1alpha1.ResourceReference) client.ObjectKey {
	return client.ObjectKey{
		Name:      res.Name,
		Namespace: res.Namespace,
	}
}
