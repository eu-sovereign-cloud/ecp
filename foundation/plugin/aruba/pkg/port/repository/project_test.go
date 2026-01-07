package repository_test

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"

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

	prj := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}

	fakeClient := newFakeProjectClientWithObject(prj)

	// Create repository
	repo := repository.NewCommonRepository[*v1alpha1.Project](fakeClient)

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
	assert.NoError(t, err, "expected Load to succeed")

	// Check that the loaded object matches the original
	assert.Equal(t, prj.Name, toLoad.Name)
	assert.Equal(t, prj.Namespace, toLoad.Namespace)
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

	updated := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}

	err = repo.Load(ctx, updated)

	assert.NoError(t, err, "expected Load to succeed")
	assert.NotNil(t, updated.Spec.Tenant)

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

	err = repo.Load(ctx, project)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

}
