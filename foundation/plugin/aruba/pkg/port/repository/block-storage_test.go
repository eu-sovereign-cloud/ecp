package repository_test

import (
	"context"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
	repo := repository.NewCommonRepository[*v1alpha1.BlockStorage](fakeClient)

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
