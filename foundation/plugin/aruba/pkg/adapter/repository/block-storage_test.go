package repository_test

import (
	"context"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kcache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/repository"
)

func newFakeStorageClientWithObject(storage *v1alpha1.BlockStorage) client.Client {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	if storage == nil {
		return fake.NewClientBuilder().WithScheme(scheme).Build()
	}

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(storage).
		Build()
}

func TestBlockStorage_Load(t *testing.T) {
	ctx := context.Background()
	// Create a fake client with one Project object
	st := &v1alpha1.BlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}

	stReference := v1alpha1.ResourceReference{
		Name:      "demo-project",
		Namespace: "default",
	}

	fakeClient := newFakeStorageClientWithObject(st)

	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctx, fakeClient, nil)

	// Prepare an empty Project object to load into
	toLoad := &v1alpha1.BlockStorage{}

	err := repo.ResolveReference(context.Background(), stReference, toLoad)
	if err != nil {
		t.Fatalf("Failed to load BlockStorage: %v", err)
	}

	if toLoad.Name != st.Name || toLoad.Namespace != st.Namespace {
		t.Fatalf("Loaded BlockStorage does not match expected values")
	}

}

func TestBlockStorage_List(t *testing.T) {
	ctx := context.Background()
	// Create a fake client with some BlockStorage objects
	st1 := &v1alpha1.BlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-1",
			Namespace: "default",
		},
	}
	st2 := &v1alpha1.BlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-2",
			Namespace: "default",
		},
	}

	fakeClient := newFakeStorageClientWithObject(nil)
	assert.NoError(t, fakeClient.Create(ctx, st1))
	assert.NoError(t, fakeClient.Create(ctx, st2))

	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctx, fakeClient, nil)

	// List BlockStorage objects
	storages, err := repo.List(ctx)
	assert.NoError(t, err, "expected List to succeed")
	assert.Len(t, storages.Items, 2, "expected 2 BlockStorage objects")
}
func TestBlockStorage_Create(t *testing.T) {
	ctx := context.Background()
	// Create a fake client with no BlockStorage objects
	fakeClient := newFakeStorageClientWithObject(nil)

	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctx, fakeClient, nil)

	// Create a new BlockStorage object
	storage := &v1alpha1.BlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "new-storage",
			Namespace: "default",
		},
	}

	err := repo.Create(ctx, storage)
	assert.NoError(t, err, "expected Create to succeed")

	// Verify that the BlockStorage was created
	loaded := &v1alpha1.BlockStorage{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "new-storage", Namespace: "default"}, loaded)
	assert.NoError(t, err, "expected to find created BlockStorage")
	assert.Equal(t, storage.Name, loaded.Name)
	assert.Equal(t, storage.Namespace, loaded.Namespace)
}

func TestBlockStorage_Update(t *testing.T) {
	ctx := context.Background()
	// Create a fake client with one BlockStorage object
	storage := &v1alpha1.BlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-storage",
			Namespace: "default",
		},
		Spec: v1alpha1.BlockStorageSpec{
			SizeGb: 10,
		},
	}
	fakeClient := newFakeStorageClientWithObject(storage)
	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctx, fakeClient, nil)

	// Update the BlockStorage object
	storage.Spec.SizeGb = 20
	err := repo.Update(ctx, storage)
	assert.NoError(t, err, "expected Update to succeed")

	// Verify that the BlockStorage was updated
	updated := &v1alpha1.BlockStorage{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "existing-storage", Namespace: "default"}, updated)
	assert.NoError(t, err, "expected to find updated BlockStorage")
	assert.Equal(t, int32(20), updated.Spec.SizeGb)
}
func TestBlockStorage_Delete(t *testing.T) {
	ctx := context.Background()
	// Create a fake client with one BlockStorage object
	storage := &v1alpha1.BlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-to-delete",
			Namespace: "default",
		},
	}
	fakeClient := newFakeStorageClientWithObject(storage)

	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctx, fakeClient, nil)

	// Delete the BlockStorage object
	err := repo.Delete(ctx, storage)
	assert.NoError(t, err, "expected Delete to succeed")

	// Verify that the BlockStorage was deleted
	deleted := &v1alpha1.BlockStorage{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "storage-to-delete", Namespace: "default"}, deleted)
	assert.Error(t, err, "expected not to find deleted BlockStorage")
}

func TestBlockStorage_Watch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	// Create a fake client with one BlockStorage object
	storage := &v1alpha1.BlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-to-watch",
			Namespace: "default",
		},
	}
	fakeClient := newFakeStorageClientWithObject(storage)
	cache := NewMockCache(mockCtrl)
	informer := NewMockInformer(mockCtrl)

	cache.EXPECT().
		GetInformer(gomock.Any(), gomock.Any()).
		Return(informer, nil).AnyTimes()

	var capturedHandler kcache.ResourceEventHandler
	informer.EXPECT().RemoveEventHandler(gomock.Any()).Return(nil).AnyTimes()

	informer.EXPECT().
		AddEventHandler(gomock.Any()).
		Do(func(handler kcache.ResourceEventHandler) {
			capturedHandler = handler

			// Simulate an update to the BlockStorage object
			updatedStorage := storage.DeepCopy()
			updatedStorage.Spec.SizeGb = 50
			assert.NoError(t, fakeClient.Update(ctx, updatedStorage), "expected Update to succeed")

			go func() {
				capturedHandler.OnUpdate(storage, updatedStorage)
			}()

		}).AnyTimes()
	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctx, fakeClient, cache)

	// Start watching the BlockStorage object
	out, cancelWatch, err := repo.Watch(ctx, storage)
	defer cancelWatch()
	assert.NoError(t, err, "expected Watch to succeed")

	// Verify that the update is received on the watch channel
	select {
	case updated := <-out:
		assert.Equal(t, int32(50), updated.Spec.SizeGb, "expected updated SizeGb to be 50")
	case <-ctx.Done():
		t.Fatal("did not receive update on watch channel")
	}
}

func TestBlockStorage_WaitUntil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	// Create a fake client with one BlockStorage object
	storage := &v1alpha1.BlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-to-wait",
			Namespace: "default",
		},
	}
	fakeClient := newFakeStorageClientWithObject(storage)

	// Create a cache
	cache := NewMockCache(mockCtrl)
	informer := NewMockInformer(mockCtrl)

	cache.EXPECT().
		GetInformer(gomock.Any(), gomock.Any()).
		Return(informer, nil).
		AnyTimes()

	informer.EXPECT().RemoveEventHandler(gomock.Any()).Return(nil).AnyTimes()

	informer.EXPECT().
		AddEventHandler(gomock.Any()).
		Do(func(handler kcache.ResourceEventHandler) {
			// Simulate an update event after a short delay
			go func() {
				// Simulate an update to the BlockStorage
				updatedStorage := storage.DeepCopy()
				updatedStorage.Spec.SizeGb = 100
				assert.NoError(t, fakeClient.Update(ctx, updatedStorage), "expected Update to succeed")
				handler.OnUpdate(storage, updatedStorage)
			}()
		}).
		AnyTimes()

	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctx, fakeClient, cache)

	out, err := repo.WaitUntil(ctx, storage, func(s *v1alpha1.BlockStorage) bool {
		return s.Spec.SizeGb == 100
	})

	assert.NoError(t, err, "expected WaitUntil to succeed")

	// Verify that the update is received
	assert.Equal(t, int32(100), out.Spec.SizeGb, "expected to receive updated BlockStorage with SizeGb 100")
}
