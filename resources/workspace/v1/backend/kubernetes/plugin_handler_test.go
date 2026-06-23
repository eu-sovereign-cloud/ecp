package kubernetes_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/workspace/v1"
	. "github.com/eu-sovereign-cloud/ecp/resources/workspace/v1/backend/kubernetes"
)

func TestWorkspacePluginHandler_HandleReconcile(t *testing.T) {
	var (
		errPlugin = errors.New("plugin failed")
		errRepo   = errors.New("repo error")
	)

	t.Run("should do nothing if resource is active", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with active state
		activeState := commondomain.ResourceStateActive
		resource := &wsdom.Workspace{
			Status: &wsdom.WorkspaceStatus{
				StatusDomain: commondomain.StatusDomain{
					State: activeState,
				},
			},
		}

		//
		// And a repo and plugin that are not expected to be called
		mockRepo := NewMockRepo[*wsdom.Workspace](ctrl)
		mockPlugin := NewMockWorkspacePlugin(ctrl)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 0)

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
		pendingState := commondomain.ResourceStatePending
		resource := &wsdom.Workspace{
			Status: &wsdom.WorkspaceStatus{
				StatusDomain: commondomain.StatusDomain{
					State: pendingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*wsdom.Workspace](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *wsdom.Workspace) (*wsdom.Workspace, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockWorkspacePlugin(ctrl)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 0)

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
		creatingState := commondomain.ResourceStateCreating
		resource := &wsdom.Workspace{
			Status: &wsdom.WorkspaceStatus{
				StatusDomain: commondomain.StatusDomain{
					State: creatingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*wsdom.Workspace](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *wsdom.Workspace) (*wsdom.Workspace, error) {
				require.Equal(t, commondomain.ResourceStateActive, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is expected to be called to create the resource
		mockPlugin := NewMockWorkspacePlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 0)

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
		// Given a resource with deleting state and a deletion timestamp
		deletingState := commondomain.ResourceStateDeleting
		now := time.Now()
		resource := &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &wsdom.WorkspaceStatus{
				StatusDomain: commondomain.StatusDomain{
					State: deletingState,
				},
			},
		}

		//
		// And a repo that is not expected to be called
		mockRepo := NewMockRepo[*wsdom.Workspace](ctrl)

		//
		// And a plugin that is expected to be called to delete the resource
		mockPlugin := NewMockWorkspacePlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 0)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and not request a requeue
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should set state to error and requeue when plugin create fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with creating state
		creatingState := commondomain.ResourceStateCreating
		resource := &wsdom.Workspace{
			Status: &wsdom.WorkspaceStatus{
				StatusDomain: commondomain.StatusDomain{
					State: creatingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to error
		mockRepo := NewMockRepo[*wsdom.Workspace](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *wsdom.Workspace) (*wsdom.Workspace, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that returns an error on create
		mockPlugin := NewMockWorkspacePlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 0)
		handler.MaxConditions = 1

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed, handle the error, and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should return error when repo update fails after plugin failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with creating state
		creatingState := commondomain.ResourceStateCreating
		resource := &wsdom.Workspace{
			Status: &wsdom.WorkspaceStatus{
				StatusDomain: commondomain.StatusDomain{
					State: creatingState,
				},
			},
		}

		//
		// And a plugin that returns an error
		mockPlugin := NewMockWorkspacePlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errPlugin)

		//
		// And a repo that returns an error on update
		mockRepo := NewMockRepo[*wsdom.Workspace](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 0)

		//
		// When we reconcile the resource
		_, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should return the repo error
		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should set state to error and requeue when plugin delete fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with deleting state and a deletion timestamp
		deletingState := commondomain.ResourceStateDeleting
		now := time.Now()
		resource := &wsdom.Workspace{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &wsdom.WorkspaceStatus{
				StatusDomain: commondomain.StatusDomain{
					State: deletingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to error
		mockRepo := NewMockRepo[*wsdom.Workspace](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *wsdom.Workspace) (*wsdom.Workspace, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that returns an error on delete
		mockPlugin := NewMockWorkspacePlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 0)
		handler.MaxConditions = 1

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed, handle the error, and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should set state to creating and requeue on retry create", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with error state that was previously creating
		errorState := commondomain.ResourceStateError
		resource := &wsdom.Workspace{
			Status: &wsdom.WorkspaceStatus{
				StatusDomain: commondomain.StatusDomain{
					State: errorState,
					Conditions: []commondomain.StatusConditionDomain{
						{State: commondomain.ResourceStatePending, LastTransitionAt: time.Now().Add(-2 * time.Minute)},
						{State: commondomain.ResourceStateCreating, LastTransitionAt: time.Now().Add(-1 * time.Minute)},
						{State: commondomain.ResourceStateError, LastTransitionAt: time.Now()},
					},
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to creating
		mockRepo := NewMockRepo[*wsdom.Workspace](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *wsdom.Workspace) (*wsdom.Workspace, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockWorkspacePlugin(ctrl)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 0)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should do nothing for unhandled states", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with an unhandled state (e.g., updating)
		updatingState := commondomain.ResourceStateUpdating
		resource := &wsdom.Workspace{
			Status: &wsdom.WorkspaceStatus{
				StatusDomain: commondomain.StatusDomain{
					State: updatingState,
				},
			},
		}

		//
		// And a repo and plugin that are not expected to be called
		mockRepo := NewMockRepo[*wsdom.Workspace](ctrl)
		mockPlugin := NewMockWorkspacePlugin(ctrl)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 0)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and do nothing
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should return error when repo update fails in setResourceState", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource in pending state
		pendingState := commondomain.ResourceStatePending
		resource := &wsdom.Workspace{
			Status: &wsdom.WorkspaceStatus{
				StatusDomain: commondomain.StatusDomain{
					State: pendingState,
				},
			},
		}

		//
		// And a repo that returns an error on update
		mockRepo := NewMockRepo[*wsdom.Workspace](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockWorkspacePlugin(ctrl)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 0)

		//
		// When we reconcile the resource
		_, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should return the repo error
		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should fatal if state changes unexpectedly after delegation", func(t *testing.T) {
		if os.Getenv("BE_FATAL") == "1" {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			creatingState := commondomain.ResourceStateCreating
			resource := &wsdom.Workspace{
				Status: &wsdom.WorkspaceStatus{
					StatusDomain: commondomain.StatusDomain{
						State: creatingState,
					},
				},
			}

			mockPlugin := NewMockWorkspacePlugin(ctrl)
			mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, res *wsdom.Workspace) error {
					// Plugin modifies the state
					res.Status.State = commondomain.ResourceStateActive
					return nil
				})

			handler := NewWorkspacePluginHandler(NewMockRepo[*wsdom.Workspace](ctrl), mockPlugin, 0)

			handler.HandleReconcile(context.Background(), resource) //nolint:errcheck
			return
		}

		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=TestWorkspacePluginHandler_HandleReconcile/should_fatal_if_state_changes_unexpectedly_after_delegation")
		cmd.Env = append(os.Environ(), "BE_FATAL=1")
		err := cmd.Run()

		if e, ok := errors.AsType[*exec.ExitError](err); ok && !e.Success() { //nolint:errorlint // acceptable for tests
			// Test passed because the subprocess exited with a non-zero status code.
			return
		}
		t.Fatalf("process ran with err %v, want exit status 1", err)
	})
}
