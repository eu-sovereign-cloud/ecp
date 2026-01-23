package handler

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

func TestBlockStoragePluginHandler_HandleReconcile(t *testing.T) {
	var (
		errPlugin = errors.New("plugin failed")
		errRepo   = errors.New("repo failed")
	)

	t.Run("should do nothing if resource is active and requires no changes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with active state and matching spec/status
		activeState := regional.ResourceStateActive
		resource := &regional.BlockStorageDomain{
			Spec: regional.BlockStorageSpec{SizeGB: 10},
			Status: &regional.BlockStorageStatus{
				State:  &activeState,
				SizeGB: 10,
			},
		}

		//
		// And a repo and plugin that are not expected to be called
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockPlugin := NewMockBlockStorage(ctrl)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

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
		pendingState := regional.ResourceStatePending
		resource := &regional.BlockStorageDomain{
			Status: &regional.BlockStorageStatus{
				State: &pendingState,
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.BlockStorageDomain) (*regional.BlockStorageDomain, error) {
				require.Equal(t, regional.ResourceStateCreating, *res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockBlockStorage(ctrl)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should set state to updating and requeue when size is increased on an active resource", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given an active resource with a spec size greater than its status size
		activeState := regional.ResourceStateActive
		resource := &regional.BlockStorageDomain{
			Spec: regional.BlockStorageSpec{SizeGB: 20},
			Status: &regional.BlockStorageStatus{
				State:  &activeState,
				SizeGB: 10,
			},
		}

		//
		// And a repo that is expected to be called once to update state to updating
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.BlockStorageDomain) (*regional.BlockStorageDomain, error) {
				require.Equal(t, regional.ResourceStateUpdating, *res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called yet
		mockPlugin := NewMockBlockStorage(ctrl)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should call plugin create and set state to active when resource is creating", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with creating state
		creatingState := regional.ResourceStateCreating
		resource := &regional.BlockStorageDomain{
			Spec: regional.BlockStorageSpec{SizeGB: 10},
			Status: &regional.BlockStorageStatus{
				State: &creatingState,
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.BlockStorageDomain) (*regional.BlockStorageDomain, error) {
				require.Equal(t, regional.ResourceStateActive, *res.Status.State)
				require.Equal(t, res.Spec.SizeGB, res.Status.SizeGB)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is expected to be called to create the resource
		mockPlugin := NewMockBlockStorage(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and not request a requeue
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should call plugin delete and set state to deleting when resource is deleting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with deleting state
		deletingState := regional.ResourceStateDeleting
		resource := &regional.BlockStorageDomain{
			Status: &regional.BlockStorageStatus{
				State: &deletingState,
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.BlockStorageDomain) (*regional.BlockStorageDomain, error) {
				require.Equal(t, regional.ResourceStateDeleting, *res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is expected to be called to delete the resource
		mockPlugin := NewMockBlockStorage(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and not request a requeue
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should call plugin increase size and set state to active when resource is updating", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with updating state and increased size
		updatingState := regional.ResourceStateUpdating
		resource := &regional.BlockStorageDomain{
			Spec: regional.BlockStorageSpec{SizeGB: 20},
			Status: &regional.BlockStorageStatus{
				State:  &updatingState,
				SizeGB: 10,
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.BlockStorageDomain) (*regional.BlockStorageDomain, error) {
				require.Equal(t, regional.ResourceStateActive, *res.Status.State)
				require.Equal(t, res.Spec.SizeGB, res.Status.SizeGB)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is expected to be called to increase size
		mockPlugin := NewMockBlockStorage(ctrl)
		mockPlugin.EXPECT().IncreaseSize(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

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
		errorState := regional.ResourceStateError
		resource := &regional.BlockStorageDomain{
			Status: &regional.BlockStorageStatus{
				State: &errorState,
				Conditions: []regional.StatusConditionDomain{
					{State: regional.ResourceStateCreating},
					{State: regional.ResourceStateError},
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to creating
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.BlockStorageDomain) (*regional.BlockStorageDomain, error) {
				require.Equal(t, regional.ResourceStateCreating, *res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockBlockStorage(ctrl)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should set state to updating and requeue on retry increase size", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with error state that was previously updating
		errorState := regional.ResourceStateError
		resource := &regional.BlockStorageDomain{
			Spec: regional.BlockStorageSpec{SizeGB: 20},
			Status: &regional.BlockStorageStatus{
				State: &errorState,
				Conditions: []regional.StatusConditionDomain{
					{State: regional.ResourceStateUpdating},
					{State: regional.ResourceStateError},
				},
				SizeGB: 10,
			},
		}

		//
		// And a repo that is expected to be called once to update state to updating
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.BlockStorageDomain) (*regional.BlockStorageDomain, error) {
				require.Equal(t, regional.ResourceStateUpdating, *res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockBlockStorage(ctrl)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

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
		creatingState := regional.ResourceStateCreating
		resource := &regional.BlockStorageDomain{
			Status: &regional.BlockStorageStatus{
				State: &creatingState,
			},
		}

		//
		// And a repo that is expected to be called once to update state to error
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.BlockStorageDomain) (*regional.BlockStorageDomain, error) {
				require.Equal(t, regional.ResourceStateError, *res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that returns an error on create
		mockPlugin := NewMockBlockStorage(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should handle the error gracefully, not return an error, but request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should return error when repo update fails after plugin failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with creating state
		creatingState := regional.ResourceStateCreating
		resource := &regional.BlockStorageDomain{
			Status: &regional.BlockStorageStatus{
				State: &creatingState,
			},
		}

		//
		// And a plugin that returns an error
		mockPlugin := NewMockBlockStorage(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errPlugin)

		//
		// And a repo that returns an error on update
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, errRepo)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

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
		pendingState := regional.ResourceStatePending
		resource := &regional.BlockStorageDomain{
			Status: &regional.BlockStorageStatus{
				State: &pendingState,
			},
		}

		//
		// And a repo that returns an error on update
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, errRepo).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockBlockStorage(ctrl)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		_, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should return the repo error
		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should fatal if state changes unexpectedly after delegation", func(t *testing.T) {
		if os.Getenv("BE_FATAL") == "1" {
			//
			// Givena controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			//
			// And a resource with creating state
			creatingState := regional.ResourceStateCreating
			resource := &regional.BlockStorageDomain{
				Status: &regional.BlockStorageStatus{
					State: &creatingState,
				},
			}

			//
			// And a plugin that modifies the resource state during delegation
			mockPlugin := NewMockBlockStorage(ctrl)
			mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, res *regional.BlockStorageDomain) error {
					activeState := regional.ResourceStateActive
					res.Status.State = &activeState
					return nil
				})

			//
			// And a block storage plugin handler
			handler := NewBlockStoragePluginHandler(NewMockRepo[*regional.BlockStorageDomain](ctrl), mockPlugin)

			//
			// When we reconcile the resource
			handler.HandleReconcile(context.Background(), resource)

			//
			// Then the process should exit with a fatal error
			return
		}

		//
		// Given a command to run the test in a separate process
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=TestBlockStoragePluginHandler_HandleReconcile/should_fatal_if_state_changes_unexpectedly_after_delegation")
		cmd.Env = append(os.Environ(), "BE_FATAL=1")

		//
		// When we run the command
		err := cmd.Run()

		//
		// Then the command should exit with a non-zero status code
		if e, ok := err.(*exec.ExitError); ok && !e.Success() { //nolint:errorlint // acceptable for tests
			return
		}
		t.Fatalf("process ran with err %v, want exit status 1", err)
	})

	t.Run("should set state to error and requeue when plugin delete fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with deleting state
		deletingState := regional.ResourceStateDeleting
		resource := &regional.BlockStorageDomain{
			Status: &regional.BlockStorageStatus{
				State: &deletingState,
			},
		}

		//
		// And a repo that is expected to be called once to update state to error
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.BlockStorageDomain) (*regional.BlockStorageDomain, error) {
				require.Equal(t, regional.ResourceStateError, *res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that returns an error on delete
		mockPlugin := NewMockBlockStorage(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should handle the error gracefully and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should set state to error and requeue when plugin increase size fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with updating state and increased size
		updatingState := regional.ResourceStateUpdating
		resource := &regional.BlockStorageDomain{
			Spec: regional.BlockStorageSpec{SizeGB: 20},
			Status: &regional.BlockStorageStatus{
				State:  &updatingState,
				SizeGB: 10,
			},
		}

		//
		// And a repo that is expected to be called once to update state to error
		mockRepo := NewMockRepo[*regional.BlockStorageDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.BlockStorageDomain) (*regional.BlockStorageDomain, error) {
				require.Equal(t, regional.ResourceStateError, *res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that returns an error on increase size
		mockPlugin := NewMockBlockStorage(ctrl)
		mockPlugin.EXPECT().IncreaseSize(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should handle the error gracefully and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})
}

func TestBlockStoragePluginHandler_HandleAdmission(t *testing.T) {
	t.Run("should succeed when size is not decreased", func(t *testing.T) {
		//
		// Given a resource with a size not being decreased
		activeState := regional.ResourceStateActive
		resource := &regional.BlockStorageDomain{
			Spec: regional.BlockStorageSpec{SizeGB: 20},
			Status: &regional.BlockStorageStatus{
				State:  &activeState,
				SizeGB: 10,
			},
		}

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(nil, nil)

		//
		// When we handle the admission of the resource
		err := handler.HandleAdmission(context.Background(), resource)

		//
		// Then it should succeed
		require.NoError(t, err)
	})

	t.Run("should succeed when size is decreased on creating", func(t *testing.T) {
		//
		// Given a resource with a size being decreased but in creating state
		creatingState := regional.ResourceStateCreating
		resource := &regional.BlockStorageDomain{
			Spec: regional.BlockStorageSpec{SizeGB: 5},
			Status: &regional.BlockStorageStatus{
				State:  &creatingState,
				SizeGB: 10,
			},
		}

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(nil, nil)

		//
		// When we handle the admission of the resource
		err := handler.HandleAdmission(context.Background(), resource)

		//
		// Then it should succeed
		require.NoError(t, err)
	})

	t.Run("should fail when size is decreased", func(t *testing.T) {
		//
		// Given a resource with a size being decreased
		activeState := regional.ResourceStateActive
		resource := &regional.BlockStorageDomain{
			Spec: regional.BlockStorageSpec{SizeGB: 5},
			Status: &regional.BlockStorageStatus{
				State:  &activeState,
				SizeGB: 10,
			},
		}

		//
		// And a block storage plugin handler
		handler := NewBlockStoragePluginHandler(nil, nil)

		//
		// When we handle the admission of the resource
		err := handler.HandleAdmission(context.Background(), resource)

		//
		// Then it should fail with a clear error message
		require.Error(t, err)
		require.Contains(t, err.Error(), "decrease storage size is not allowed")
	})
}
