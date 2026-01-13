package repository_test

import (
	"context"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	generic_repository "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/repository"
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

	prj := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}

	fakeClient := newFakeProjectClientWithObject(prj)

	// Create repository
	repo := generic_repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](fakeClient, &v1alpha1.ProjectList{})

	// Prepare an empty Project object to load into
	toLoad := &v1alpha1.Project{}

	// load the Project via a BlockStorage's ProjectReference
	bs := &v1alpha1.BlockStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-project",
			Namespace: "default",
		},

		Spec: v1alpha1.BlockStorageSpec{
			ProjectReference: v1alpha1.ResourceReference{
				Name:      "demo-project",
				Namespace: "default",
			},
		},
	}
	err := repo.ResolveReference(ctx, bs.Spec.ProjectReference, toLoad)
	require.NoError(t, err, "expected Load to succeed")

	// Check that the loaded object matches the original
	require.Equal(t, prj.Name, toLoad.Name)
	require.Equal(t, prj.Namespace, toLoad.Namespace)
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
	repo := generic_repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](fakeClient, &v1alpha1.ProjectList{})

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
	repo := generic_repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](fakeClient, &v1alpha1.ProjectList{})

	project.Spec.Tenant = "tenant"
	err := repo.Update(ctx, project)
	require.NoError(t, err, "expected Load to succeed")

	updated := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}

	err = repo.Load(ctx, updated)

	require.NoError(t, err, "expected Load to succeed")
	require.NotNil(t, updated.Spec.Tenant)

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
	repo := generic_repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](fakeClient, &v1alpha1.ProjectList{})

	err := repo.Delete(ctx, project)
	require.NoError(t, err, "expected Load to succeed")

	err = repo.Load(ctx, project)
	require.Error(t, err)
	require.True(t, errors.IsNotFound(err))

}

func TestProjectRepository_List(t *testing.T) {
	ctx := context.Background()
	// Create a fake client with one Project object
	project1 := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-project-1",
			Namespace: "default",
		},
	}
	project2 := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-project-2",
			Namespace: "default",
		},
	}

	fakeClient := newFakeProjectClientWithObject(nil)
	assert.NoError(t, fakeClient.Create(ctx, project1))
	assert.NoError(t, fakeClient.Create(ctx, project2))

	// Create repository
	repo := generic_repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](fakeClient, &v1alpha1.ProjectList{})

	res, err := repo.List(ctx, client.InNamespace("default"))
	assert.NoError(t, err, "expected List to succeed")
	assert.Len(t, res, 2, "expected to list 2 projects")
}

func TestProjectRepository_WaitUntil(t *testing.T) {
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
	repo := generic_repository.NewGenericRepository[*v1alpha1.Project](fakeClient, &v1alpha1.ProjectList{})

	watchCtx, _ := context.WithCancel(ctx)
	updatedProject := project.DeepCopy()
	updatedProject.Spec.Description = "Updated description"
	err := fakeClient.Update(ctx, updatedProject)
	assert.NoError(t, err, "expected Update to succeed")
	out, err := repo.WaitUntil(watchCtx, project, func(p *v1alpha1.Project) bool {
		return p.Spec.Description == "Updated description"
	})
	assert.NoError(t, err, "expected WaitUntil to succeed")
	// Simulate an update to the project

	// Wait for the update to be received
	assert.Equal(t, "Updated description", out.Spec.Description, "expected to receive updated project")
}

func TestGenericRepository_Watch(t *testing.T) {
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
	repo := generic_repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](fakeClient, &v1alpha1.ProjectList{})

	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	out, cancelWatch, err := repo.Watch(watchCtx, project)
	assert.NoError(t, err, "expected Watch to succeed")
	defer cancelWatch()

	// Simulate an update to the project
	updatedProject := project.DeepCopy()
	updatedProject.Spec.Description = "Updated description"
	err = fakeClient.Update(ctx, updatedProject)
	assert.NoError(t, err, "expected Update to succeed")

	// Wait for the update to be received
	received := <-out
	assert.Equal(t, "Updated description", received.Spec.Description, "expected to receive updated project")
}
