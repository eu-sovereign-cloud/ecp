package repository

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	kcache "k8s.io/client-go/tools/cache"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository"
)

// GenericRepository is a generic repository for Kubernetes resources.
// It supports basic CRUD operations, listing, watching, and waiting for conditions.
// T represents the resource type (e.g., Deployment), L represents the resource list type (e.g., DeploymentList).
type GenericRepository[T client.Object, L client.ObjectList] struct {
	client client.Client
	cache  crcache.Cache

	informer     crcache.Informer
	informerOnce *sync.Once
}

// NewGenericRepository creates a new instance of GenericRepository.
func NewGenericRepository[T client.Object, L client.ObjectList](_ context.Context, client client.Client, cache crcache.Cache) *GenericRepository[T, L] {
	return &GenericRepository[T, L]{client: client, cache: cache, informerOnce: &sync.Once{}}
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
	var informerErr error
	r.informerOnce.Do(func() { //nolint:contextcheck // We want the informer to be alive after the call end.
		if r.informer == nil {
			objType := reflect.New(reflect.TypeOf(resource).Elem()).Interface().(T)
			// The informer is shared across all Watch calls for this repository instance.
			// Its lifecycle should not be tied to the context of the first Watch() call.
			// Using context.Background() makes it live for the duration of the application,
			// which is a typical lifecycle for a shared informer.
			r.informer, informerErr = r.cache.GetInformer(context.Background(), objType)
		}
	})
	if informerErr != nil {
		fmt.Printf("error getting informer: %v\n", informerErr) // TODO: replace for logger
		r.informer = nil
		r.informerOnce = &sync.Once{}

		return nil, nil, informerErr
	}

	ictx, cancel := context.WithCancel(ctx)
	out := make(chan T)
	var wg sync.WaitGroup

	handler, err := r.informer.AddEventHandler(&kcache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			r.handle(ictx, obj, resource, out, &wg)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if oldObj.(client.Object).GetResourceVersion() == newObj.(client.Object).GetResourceVersion() {
				fmt.Printf("Resource version unchanged for object %+v, skipping update handling\n", newObj)
				return
			}
			r.handle(ictx, newObj, resource, out, &wg)
		},
		DeleteFunc: func(obj interface{}) {
			r.handle(ictx, obj, resource, out, &wg)
		},
	})
	if err != nil {
		fmt.Printf("error adding handler to informer: %v\n", err) // TODO: replace for logger
		cancel()
		return nil, nil, err
	}

	go func() {
		// 1. Wait for the watch to be canceled.
		<-ictx.Done()

		// 2. Wait for any in-flight handle() calls to finish. They will either
		//    complete the send or abort because the context is canceled.
		wg.Wait()

		// 3. Now that no handlers are trying to send, it's safe to remove the
		//    handler and close the channel.
		r.informer.RemoveEventHandler(handler) //nolint:errcheck // TODO: better error handling
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

// handle processes an event object and sends it to the output channel if it matches the filter
func (r *GenericRepository[T, L]) handle(ctx context.Context, obj interface{}, filter T, out chan T, wg *sync.WaitGroup) {
	// Don't process anything if the context is already canceled.
	if ctx.Err() != nil {
		return
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

	// Increment WaitGroup counter before attempting to send.
	// This ensures that the cleanup process will wait for this goroutine to finish.
	wg.Add(1)
	defer wg.Done()

	// Use a select statement to attempt the send. If the context is canceled
	// while waiting, the send is aborted, preventing an indefinite block.
	select {
	case out <- copied:
		// Sent successfully.
	case <-ctx.Done():
		// Context was canceled, abort the send.
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
