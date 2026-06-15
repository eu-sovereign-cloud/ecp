package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	model "github.com/eu-sovereign-cloud/ecp/foundation/models"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
)

func TestImagePluginHandler_HandleReconcile(t *testing.T) {
	var (
		errPlugin = errors.New("plugin failed")
		errRepo   = errors.New("repo failed")
	)

	t.Run("should do nothing when resource is active", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given an active resource
		resource := &regional.ImageDomain{
			Status: &regional.ImageStatusDomain{
				StatusDomain: regional.StatusDomain{State: regional.ResourceStateActive},
			},
		}

		//
		// And a repo and plugin that are not expected to be called
		mockRepo := NewMockRepo[*regional.ImageDomain](ctrl)
		mockPlugin := NewMockImage(ctrl)

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and not request a requeue
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should set state to pending when resource is accepted", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a freshly accepted resource (no status yet)
		resource := &regional.ImageDomain{}

		//
		// And a repo expected to be called once to update state to pending
		mockRepo := NewMockRepo[*regional.ImageDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.ImageDomain) (*regional.ImageDomain, error) {
				require.Equal(t, regional.ResourceStatePending, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockImage(ctrl)

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and not request a requeue
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should set state to creating and requeue when resource is pending", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with pending state
		resource := &regional.ImageDomain{
			Status: &regional.ImageStatusDomain{
				StatusDomain: regional.StatusDomain{State: regional.ResourceStatePending},
			},
		}

		//
		// And a repo expected to be called once to update state to creating
		mockRepo := NewMockRepo[*regional.ImageDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.ImageDomain) (*regional.ImageDomain, error) {
				require.Equal(t, regional.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockImage(ctrl)

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should call plugin create, set size and state to active when resource is creating", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with creating state
		resource := &regional.ImageDomain{
			Status: &regional.ImageStatusDomain{
				StatusDomain: regional.StatusDomain{State: regional.ResourceStateCreating},
			},
		}

		//
		// And a repo expected to be called once to update state to active with size set
		mockRepo := NewMockRepo[*regional.ImageDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.ImageDomain) (*regional.ImageDomain, error) {
				require.Equal(t, regional.ResourceStateActive, res.Status.State)
				require.NotNil(t, res.Status.SizeMB)
				require.Equal(t, defaultImageSizeMB, *res.Status.SizeMB)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is expected to be called to create the resource
		mockPlugin := NewMockImage(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and not request a requeue
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should set state to deleting and requeue when resource is marked for deletion", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given an active resource with a deletion timestamp
		now := time.Now()
		resource := &regional.ImageDomain{
			Metadata: regional.Metadata{
				CommonMetadata: model.CommonMetadata{DeletedAt: &now},
			},
			Status: &regional.ImageStatusDomain{
				StatusDomain: regional.StatusDomain{State: regional.ResourceStateActive},
			},
		}

		//
		// And a repo expected to be called once to update state to deleting
		mockRepo := NewMockRepo[*regional.ImageDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.ImageDomain) (*regional.ImageDomain, error) {
				require.Equal(t, regional.ResourceStateDeleting, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called yet
		mockPlugin := NewMockImage(ctrl)

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should call plugin delete when resource is deleting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with deleting state and a deletion timestamp
		now := time.Now()
		resource := &regional.ImageDomain{
			Metadata: regional.Metadata{
				CommonMetadata: model.CommonMetadata{DeletedAt: &now},
			},
			Status: &regional.ImageStatusDomain{
				StatusDomain: regional.StatusDomain{State: regional.ResourceStateDeleting},
			},
		}

		//
		// And a repo that is not expected to be called
		mockRepo := NewMockRepo[*regional.ImageDomain](ctrl)

		//
		// And a plugin that is expected to be called to delete the resource
		mockPlugin := NewMockImage(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and not request a requeue
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should set state to creating and requeue on retry create", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with error state that was previously creating
		resource := &regional.ImageDomain{
			Status: &regional.ImageStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: regional.ResourceStateError,
					Conditions: []regional.StatusConditionDomain{
						{State: regional.ResourceStateCreating},
						{State: regional.ResourceStateError},
					},
				},
			},
		}

		//
		// And a repo expected to be called once to update state to creating
		mockRepo := NewMockRepo[*regional.ImageDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.ImageDomain) (*regional.ImageDomain, error) {
				require.Equal(t, regional.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockImage(ctrl)

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should set state to error and requeue when plugin create fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with creating state
		resource := &regional.ImageDomain{
			Status: &regional.ImageStatusDomain{
				StatusDomain: regional.StatusDomain{State: regional.ResourceStateCreating},
			},
		}

		//
		// And a repo expected to be called once to update state to error
		mockRepo := NewMockRepo[*regional.ImageDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.ImageDomain) (*regional.ImageDomain, error) {
				require.Equal(t, regional.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that returns an error on create
		mockPlugin := NewMockImage(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(mockRepo, mockPlugin)
		handler.MaxConditions = 1

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should handle the error gracefully and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should return error when repo update fails after plugin failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with creating state
		resource := &regional.ImageDomain{
			Status: &regional.ImageStatusDomain{
				StatusDomain: regional.StatusDomain{State: regional.ResourceStateCreating},
			},
		}

		//
		// And a plugin that returns an error
		mockPlugin := NewMockImage(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errPlugin)

		//
		// And a repo that returns an error on update
		mockRepo := NewMockRepo[*regional.ImageDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo)

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		_, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should return the repo error
		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should return error when setResourceState fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource in pending state
		resource := &regional.ImageDomain{
			Status: &regional.ImageStatusDomain{
				StatusDomain: regional.StatusDomain{State: regional.ResourceStatePending},
			},
		}

		//
		// And a repo that returns an error on update
		mockRepo := NewMockRepo[*regional.ImageDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockImage(ctrl)

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		_, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should return the repo error
		require.ErrorIs(t, err, errRepo)
	})
}

func TestImagePluginHandler_HandleAdmission(t *testing.T) {
	t.Run("should allow creation of a new image", func(t *testing.T) {
		//
		// Given a fresh image (no status yet)
		resource := &regional.ImageDomain{
			Spec: regional.ImageSpecDomain{
				CpuArchitecture: "amd64",
				Boot:            "UEFI",
			},
		}

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(nil, nil)

		//
		// When we handle the admission of the resource
		err := handler.HandleAdmission(context.Background(), resource)

		//
		// Then it should succeed
		require.NoError(t, err)
	})

	t.Run("should allow updates while the image is still being created", func(t *testing.T) {
		//
		// Given an image that has not finished creating
		resource := &regional.ImageDomain{
			Status: &regional.ImageStatusDomain{
				StatusDomain: regional.StatusDomain{State: regional.ResourceStateCreating},
			},
		}

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(nil, nil)

		//
		// When we handle the admission of the resource
		err := handler.HandleAdmission(context.Background(), resource)

		//
		// Then it should succeed
		require.NoError(t, err)
	})

	t.Run("should reject mutating an already-created image", func(t *testing.T) {
		//
		// Given an image that has already been created (active)
		resource := &regional.ImageDomain{
			Status: &regional.ImageStatusDomain{
				StatusDomain: regional.StatusDomain{State: regional.ResourceStateActive},
			},
		}

		//
		// And an image plugin handler
		handler := NewImagePluginHandler(nil, nil)

		//
		// When we handle the admission of the resource
		err := handler.HandleAdmission(context.Background(), resource)

		//
		// Then it should fail with a clear immutability error
		require.Error(t, err)
		require.Contains(t, err.Error(), "immutable")
	})
}
