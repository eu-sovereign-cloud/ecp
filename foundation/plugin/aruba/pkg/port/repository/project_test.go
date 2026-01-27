package repository_test

import (
	"context"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository"
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
	require.NoError(t, err, "expected Load to succeed")

	// Check that the loaded object matches the original
	require.Equal(t, project.Name, toLoad.Name)
	require.Equal(t, project.Namespace, toLoad.Namespace)
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
	require.NoError(t, err, "expected Load to succeed")

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
	require.NoError(t, err, "expected Load to succeed")
	require.NotNil(t, project.Spec.Tenant)

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
	require.NoError(t, err, "expected Load to succeed")

}
