package repository_test

import (
	context "context"
	"testing"
	"time"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/generic/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kcache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
		ObjectMeta: v1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}

	fakeClient := newFakeProjectClientWithObject(prj)

	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctx, fakeClient, nil)

	// Prepare an empty Project object to load into
	toLoad := &v1alpha1.Project{}

	// load the Project via a BlockStorage's ProjectReference
	bs := &v1alpha1.BlockStorage{
		ObjectMeta: v1.ObjectMeta{
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
		ObjectMeta: v1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}
	fakeClient := newFakeProjectClientWithObject(nil)

	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctx, fakeClient, nil)

	err := repo.Create(ctx, project)
	require.NoError(t, err, "expected Load to succeed")

}

func TestProjectRepository_Update(t *testing.T) {
	ctx := context.Background()
	// Create a fake client with one Project object
	project := &v1alpha1.Project{
		ObjectMeta: v1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}
	fakeClient := newFakeProjectClientWithObject(project)

	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctx, fakeClient, nil)

	project.Spec.Tenant = "tenant"
	err := repo.Update(ctx, project)
	require.NoError(t, err, "expected Load to succeed")

	updated := &v1alpha1.Project{
		ObjectMeta: v1.ObjectMeta{
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
		ObjectMeta: v1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}
	fakeClient := newFakeProjectClientWithObject(project)

	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctx, fakeClient, nil)

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
		ObjectMeta: v1.ObjectMeta{
			Name:      "demo-project-1",
			Namespace: "default",
		},
	}
	project2 := &v1alpha1.Project{
		ObjectMeta: v1.ObjectMeta{
			Name:      "demo-project-2",
			Namespace: "default",
		},
	}

	fakeClient := newFakeProjectClientWithObject(nil)
	assert.NoError(t, fakeClient.Create(ctx, project1))
	assert.NoError(t, fakeClient.Create(ctx, project2))

	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctx, fakeClient, nil)

	res, err := repo.List(ctx, client.InNamespace("default"))
	assert.NoError(t, err, "expected List to succeed")
	assert.Len(t, res.Items, 2, "expected to find 2 projects")

}

func TestProjectRepository_WaitUntil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	// Create a fake client with one Project object
	project := &v1alpha1.Project{
		ObjectMeta: v1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}
	fakeClient := newFakeProjectClientWithObject(project)

	// Create a cache
	cache := NewMockCache(mockCtrl)
	informer := NewMockInformer(mockCtrl)
	informer.EXPECT().RemoveEventHandler(gomock.Any()).Return(nil).AnyTimes()

	cache.EXPECT().
		GetInformer(gomock.Any(), gomock.Any()).
		Return(informer, nil).
		AnyTimes()

	informer.EXPECT().
		AddEventHandler(gomock.Any()).
		Do(func(handler kcache.ResourceEventHandler) {
			// Simulate an update event after a short delay
			go func() {
				// Simulate an update to the project
				updatedProject := project.DeepCopy()
				updatedProject.Spec.Description = "Updated description"
				assert.NoError(t, fakeClient.Update(ctx, updatedProject), "expected Update to succeed")

				handler.OnUpdate(project, updatedProject)
			}()
		}).
		AnyTimes()

	// Create repository
	repo := repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctx, fakeClient, cache)

	out, err := repo.WaitUntil(ctx, project, func(p *v1alpha1.Project) bool {
		return p.Spec.Description == "Updated description"
	})
	assert.NoError(t, err, "expected WaitUntil to succeed")

	// Wait for the update to be received
	assert.Equal(t, "Updated description", out.Spec.Description, "expected to receive updated project")
}

// TestGenericRepository_Watch_WithMockCache tests the Watch method of GenericRepository using a mock Cache and Informer.
func TestGenericRepository_WatchWithMockCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	project := &v1alpha1.Project{
		ObjectMeta: v1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}

	fakeClient := newFakeProjectClientWithObject(project)
	mockCache := NewMockCache(ctrl)
	mockInformer := NewMockInformer(ctrl)

	var capturedHandler kcache.ResourceEventHandler
	mockCache.EXPECT().
		GetInformer(gomock.Any(), gomock.Any()).
		Return(mockInformer, nil).
		AnyTimes()
	mockInformer.EXPECT().RemoveEventHandler(gomock.Any()).Return(nil).AnyTimes()

	mockInformer.EXPECT().
		AddEventHandler(gomock.Any()).
		Do(func(handler kcache.ResourceEventHandler) {
			updatedProject := project.DeepCopy()
			updatedProject.Spec.Description = "Updated description"
			capturedHandler = handler

			// Simulate an update event after a short delay
			go func() {
				// Simulate an update to the project
				handler.OnUpdate(project, updatedProject)
			}()
		}).
		AnyTimes()

	repo := repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctx,
		fakeClient,
		mockCache,
	)

	out, cancelWatch, err := repo.Watch(ctx, project)
	defer cancelWatch()
	require.NoError(t, err)

	require.NotNil(t, capturedHandler)

	// capturedHandler.OnUpdate(project, updated)

	select {
	case received := <-out:
		assert.Equal(t, "demo-project", received.Name)
		assert.Equal(t, "Updated description", received.Spec.Description)
	case <-time.After(2 * time.Second):
		t.Fatal("expected watch event")
	}
}

// TestGenericRepository_WatchWithMockCache_FailToMatch tests the Watch method of GenericRepository using a mock Cache and Informer,
// ensuring that events that do not match the filter are ignored.
func TestGenericRepository_WatchWithMockCache_FailToMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	project := &v1alpha1.Project{
		ObjectMeta: v1.ObjectMeta{
			Name:      "demo-project",
			Namespace: "default",
		},
	}

	fakeClient := newFakeProjectClientWithObject(project)
	mockCache := NewMockCache(ctrl)
	mockInformer := NewMockInformer(ctrl)

	var capturedHandler kcache.ResourceEventHandler
	mockCache.EXPECT().
		GetInformer(gomock.Any(), gomock.Any()).
		Return(mockInformer, nil).
		AnyTimes()
	mockInformer.EXPECT().RemoveEventHandler(gomock.Any()).Return(nil).AnyTimes()

	mockInformer.EXPECT().
		AddEventHandler(gomock.Any()).
		Do(func(handler kcache.ResourceEventHandler) {
			updatedProject := project.DeepCopy()
			updatedProject.Spec.Description = "Updated description"
			capturedHandler = handler

			// Simulate an update event after a short delay
			go func() {
				// Simulate an update to the project
				handler.OnUpdate(project, updatedProject)
			}()
		}).
		AnyTimes()

	repo := repository.NewGenericRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctx,
		fakeClient,
		mockCache,
	)

	out, cancelWatch, err := repo.Watch(ctx, &v1alpha1.Project{
		ObjectMeta: v1.ObjectMeta{
			Name:      "non-matching-project",
			Namespace: "default",
		},
	})
	defer cancelWatch()
	require.NoError(t, err)

	require.NotNil(t, capturedHandler)

	select {
	case <-out:
		t.Fatal("did not expect to receive a watch event")
	case <-time.After(1 * time.Second):
		// expected timeout
	}
}
