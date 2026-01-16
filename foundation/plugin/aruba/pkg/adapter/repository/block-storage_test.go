package repository_test

import (
	"context"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	generic_repository "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/repository"
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
	_ = context.Background()
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
	repo := generic_repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](fakeClient, &v1alpha1.BlockStorageList{})

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
	repo := generic_repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](fakeClient, &v1alpha1.BlockStorageList{})

	// List BlockStorage objects
	storages, err := repo.List(ctx)
	assert.NoError(t, err, "expected List to succeed")
	assert.Len(t, storages, 2, "expected 2 BlockStorage objects")
}
func TestBlockStorage_Create(t *testing.T) {
	ctx := context.Background()
	// Create a fake client with no BlockStorage objects
	fakeClient := newFakeStorageClientWithObject(nil)

	// Create repository
	repo := generic_repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](fakeClient, &v1alpha1.BlockStorageList{})

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
	repo := generic_repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](fakeClient, &v1alpha1.BlockStorageList{})

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
	repo := generic_repository.NewGenericRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](fakeClient, &v1alpha1.BlockStorageList{})

	// Delete the BlockStorage object
	err := repo.Delete(ctx, storage)
	assert.NoError(t, err, "expected Delete to succeed")

	// Verify that the BlockStorage was deleted
	deleted := &v1alpha1.BlockStorage{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "storage-to-delete", Namespace: "default"}, deleted)
	assert.Error(t, err, "expected not to find deleted BlockStorage")
}
