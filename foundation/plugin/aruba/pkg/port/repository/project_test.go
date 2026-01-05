package repository_test

import (
	"context"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newFakeProjectClientWithObject(project *v1alpha1.Project) client.Client {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	if project == nil {
		return fake.NewClientBuilder().WithScheme(scheme).Build()
	}

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		Build()
}

func TestProjectRepository_Load(t *testing.T) {
	ctx := context.Background()
	// Create a fake client with one Project object
	project := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}
	fakeClient := newFakeProjectClientWithObject(project)

	// Create repository
	repo := repository.NewCommonRepository[*v1alpha1.Project](fakeClient)

	// Prepare an empty Project object to load into
	toLoad := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}

	// Call Load
	err := repo.Load(ctx, toLoad)
	assert.NoError(t, err, "expected Load to succeed")

	// Check that the loaded object matches the original
	assert.Equal(t, project.Name, toLoad.Name)
	assert.Equal(t, project.Namespace, toLoad.Namespace)
}

func TestProjectRepository_Create(t *testing.T) {
	ctx := context.Background()
	// Create a fake client with one Project object
	project := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}
	fakeClient := newFakeProjectClientWithObject(nil)

	// Create repository
	repo := repository.NewCommonRepository[*v1alpha1.Project](fakeClient)

	err := repo.Create(ctx, project)
	assert.NoError(t, err, "expected Load to succeed")

}

func TestProjectRepository_Update(t *testing.T) {
	ctx := context.Background()
	// Create a fake client with one Project object
	project := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}
	fakeClient := newFakeProjectClientWithObject(project)

	// Create repository
	repo := repository.NewCommonRepository[*v1alpha1.Project](fakeClient)

	project.Spec.Tenant = "tenant"
	err := repo.Update(ctx, project)
	assert.NoError(t, err, "expected Load to succeed")
	assert.NotNil(t, project.Spec.Tenant)

}

func TestProjectRepository_Delete(t *testing.T) {
	ctx := context.Background()
	// Create a fake client with one Project object
	project := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}
	fakeClient := newFakeProjectClientWithObject(project)

	// Create repository
	repo := repository.NewCommonRepository[*v1alpha1.Project](fakeClient)

	err := repo.Delete(ctx, project)
	assert.NoError(t, err, "expected Load to succeed")

}
