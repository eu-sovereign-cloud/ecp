package repository

import (
	"context"
)

// CLUDFunc is a generic function for CLUD (Create, Load, Update and Delete)
// operations with instances of a given type.
//
// Important: type T must always be a reference (pointer) for the underlying
// entity model to ensure consistent behavior across all repository operations.
//
// Important: because CLUD functions can mutate data on resources references
// passed as parameters, please deepcopy them if you need to keep the original
// data.
type CLUDFunc[T any] func(ctx context.Context, resource T) error

// ListFunc is a generic function for listing instances of a given type from a
// repository.
//
// Important: type T must always be a reference (pointer) for the underlying
// entity model to ensure consistent behavior across all repository operations.
//
// Important: because List functions can mutate data on resources references
// passed as parameters, please deepcopy them if you need to keep the original
// data.
type ListFunc[T any] func(ctx context.Context, resource T) ([]T, error)

// WatchFunc is a generic function for watching changes to instances of a given
// type in a repository.
//
// Important: type T must always be a reference (pointer) for the underlying
// entity model to ensure consistent behavior across all repository operations.
type WatchFunc[T any] func(ctx context.Context, resource T) (out chan T, cancel func(), err error)

// WaitUntilFunc is a generic function for watching changes to instances of a
// given type in a repository until a condition is met.
//
// Important: type T must always be a reference (pointer) for the underlying
// entity model to ensure consistent behavior across all repository operations.
type WaitUntilFunc[T any] func(ctx context.Context, resource T, condition WaitConditionFunc[T]) (T, error)

// WaitConditionFunc is a function that returns true if a given resource
// matches a condition.
//
// Important: type T must always be a reference (pointer) for the underlying
// entity model to ensure consistent behavior across all repository operations.
type WaitConditionFunc[T any] func(resource T) bool

// Repository is a generic interface for a repository of resources.
// It combines the Reader, Writer, and Watcher interfaces.
//
// Important: type T must always be a reference (pointer) for the underlying
// entity model to ensure consistent behavior across all repository operations.
//
// Important: because CLUD functions can mutate data on resources references
// passed as parameters, please deepcopy them if you need to keep the original
// data.
type Repository[T any] interface {
	Reader[T]
	Writer[T]
	Watcher[T]
}

// Reader is a generic interface for reading resources from a repository.
//
// Important: type T must always be a reference (pointer) for the underlying
// entity model to ensure consistent behavior across all repository operations.
//
// Important: because CLUD functions can mutate data on resources references
// passed as parameters, please deepcopy them if you need to keep the original
// data.
type Reader[T any] interface {
	// Load retrieves a single resource from the repository.
	//
	// The resource to be loaded is identified by the fields set in the
	// provided 'resource' argument.
	Load(ctx context.Context, resource T) error
	// List retrieves a list of resources from the repository.
	//
	// The resources to be listed can be filtered by the fields set in the
	// provided 'resource' argument.
	List(ctx context.Context, resource T) ([]T, error)
}

// Writer is a generic interface for writing resources to a repository.
//
// Important: type T must always be a reference (pointer) for the underlying
// entity model to ensure consistent behavior across all repository operations.
//
// Important: because CLUD functions can mutate data on resources references
// passed as parameters, please deepcopy them if you need to keep the original
// data.
type Writer[T any] interface {
	// Create adds a new resource to the repository.
	Create(ctx context.Context, resource T) error
	// Update modifies an existing resource in the repository.
	Update(ctx context.Context, resource T) error
	// Delete removes a resource from the repository.
	Delete(ctx context.Context, resource T) error
}

// Watcher is a generic interface for watching for changes to resources in a
// repository.
type Watcher[T any] interface {
	// Watch monitors a resource for changes.
	// It returns a channel that will receive the updated resource whenever a
	// change is detected.
	//
	// The 'resource' argument specifies which resource to watch, filtering by
	// the fields that are set.
	//
	// The method also returns a 'cancel' function that must be called to stop
	// watching for changes and clean up resources.
	//
	// Finally, it can return an error if setting up the watch fails.
	//
	// Example usage:
	//   out, cancel, err := watcher.Watch(ctx, myResource)
	//   if err != nil {
	//     // handle error
	//   }
	//   defer cancel()
	//
	//   updateCount := 0
	//   for {
	//     select {
	//     case updatedResource := <-out:
	//       // Process updated resource
	//       updateCount++
	//       // Example condition: stop after 5 updates
	//       if updateCount >= 5 {
	//         // It will result on calling the deferred cancel to stop the
	//         // watch
	//         return
	//       }
	//     case <-ctx.Done():
	//       // context was cancelled
	//       return
	//     }
	//   }
	Watch(ctx context.Context, resource T) (out chan T, cancel func(), err error)

	// WaitUntil monitors a resource for changes until a condition is met.
	//
	// The 'resource' argument specifies which resource to watch, filtering by
	// the fields that are set.
	//
	// The 'condition' argument is a function that will be called with each
	// update of the resource. The watch will continue until the condition
	// returns true.
	//
	// It will return the first version of the resource which matches the
	// condition.
	//
	// Finally, it can return an error if setting up the watch fails.
	//
	// Example usage:
	//   condition := func(r *MyResource) bool {
	//     return r.Status == "Ready"
	//   }
	//   updatedResource, err := watcher.WaitUntil(ctx, myResource, condition)
	//   if err != nil {
	//     // handle error
	//   }
	WaitUntil(ctx context.Context, resource T, condition WaitConditionFunc[T]) (T, error)
}
