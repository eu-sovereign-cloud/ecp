package handler

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

func TestWorkspacePluginHandler_HandleReconcile(t *testing.T) {
	var (
		errPlugin = errors.New("plugin failed")
	)

	t.Run("should do nothing if resource state is active", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with active state
		activeState := regional.ResourceStateActive
		resource := &regional.WorkspaceDomain{
			Status: regional.WorkspaceStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: &activeState,
				},
			},
		}

		//
		// And a repo and plugin that are not expected to be called
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)
		mockPlugin := NewMockWorkspace(ctrl)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed
		require.NoError(t, err)
	})

	t.Run("should set state to creating when resource is pending", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with pending state
		pendingState := regional.ResourceStatePending
		resource := &regional.WorkspaceDomain{
			Status: regional.WorkspaceStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: &pendingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.WorkspaceDomain) (*regional.WorkspaceDomain, error) {
				require.Equal(t, regional.ResourceStateCreating, *res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockWorkspace(ctrl)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed
		require.NoError(t, err)
	})

	t.Run("should call plugin create and set state to active when resource is creating", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with creating state
		creatingState := regional.ResourceStateCreating
		resource := &regional.WorkspaceDomain{
			Status: regional.WorkspaceStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: &creatingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.WorkspaceDomain) (*regional.WorkspaceDomain, error) {
				require.Equal(t, regional.ResourceStateActive, *res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is expected to be called to create the resource
		mockPlugin := NewMockWorkspace(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed
		require.NoError(t, err)
	})

	t.Run("should call plugin delete and set state to deleting when resource is deleting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with deleting state
		deletingState := regional.ResourceStateDeleting
		resource := &regional.WorkspaceDomain{
			Status: regional.WorkspaceStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: &deletingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.WorkspaceDomain) (*regional.WorkspaceDomain, error) {
				require.Equal(t, regional.ResourceStateDeleting, *res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is expected to be called to delete the resource
		mockPlugin := NewMockWorkspace(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed
		require.NoError(t, err)
	})

	t.Run("should set state to error when plugin create fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with creating state
		creatingState := regional.ResourceStateCreating
		resource := &regional.WorkspaceDomain{
			Status: regional.WorkspaceStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: &creatingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to error
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.WorkspaceDomain) (*regional.WorkspaceDomain, error) {
				require.Equal(t, regional.ResourceStateError, *res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that returns an error on create
		mockPlugin := NewMockWorkspace(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed (error is handled by updating status)
		require.NoError(t, err)
	})

	t.Run("should return error when repo update fails after plugin failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with creating state
		creatingState := regional.ResourceStateCreating
		resource := &regional.WorkspaceDomain{
			Status: regional.WorkspaceStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: &creatingState,
				},
			},
		}

		//
		// And a plugin that returns an error
		errPlugin := errors.New("plugin error")
		mockPlugin := NewMockWorkspace(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errPlugin)

		//
		// And a repo that returns an error on update
		errRepo := errors.New("repo error")
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, errRepo)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should return the repo error
		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should set state to error when plugin delete fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with deleting state
		deletingState := regional.ResourceStateDeleting
		resource := &regional.WorkspaceDomain{
			Status: regional.WorkspaceStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: &deletingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to error
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.WorkspaceDomain) (*regional.WorkspaceDomain, error) {
				require.Equal(t, regional.ResourceStateError, *res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that returns an error on delete
		mockPlugin := NewMockWorkspace(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed (error is handled by updating status)
		require.NoError(t, err)
	})

	t.Run("should set state to creating on retry create", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with error state that was previously creating
		errorState := regional.ResourceStateError
		resource := &regional.WorkspaceDomain{
			Status: regional.WorkspaceStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: &errorState,
					Conditions: []regional.StatusConditionDomain{
						{State: regional.ResourceStatePending, LastTransitionAt: time.Now().Add(-2 * time.Minute)},
						{State: regional.ResourceStateCreating, LastTransitionAt: time.Now().Add(-1 * time.Minute)},
						{State: regional.ResourceStateError, LastTransitionAt: time.Now()},
					},
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to creating
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.WorkspaceDomain) (*regional.WorkspaceDomain, error) {
				require.Equal(t, regional.ResourceStateCreating, *res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockWorkspace(ctrl)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed
		require.NoError(t, err)
	})

	t.Run("should do nothing for unhandled states", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with an unhandled state (e.g., updating)
		updatingState := regional.ResourceStateUpdating
		resource := &regional.WorkspaceDomain{
			Status: regional.WorkspaceStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: &updatingState,
				},
			},
		}

		//
		// And a repo and plugin that are not expected to be called
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)
		mockPlugin := NewMockWorkspace(ctrl)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and do nothing
		require.NoError(t, err)
	})

	t.Run("should return error when repo update fails in setResourceState", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource in pending state
		pendingState := regional.ResourceStatePending
		resource := &regional.WorkspaceDomain{
			Status: regional.WorkspaceStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: &pendingState,
				},
			},
		}

		//
		// And a repo that returns an error on update
		errRepo := errors.New("repo update error")
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, errRepo).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockWorkspace(ctrl)

		//
		// And a workspace plugin handler
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin)

		//
		// When we reconcile the resource
		err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should return the repo error
		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should fatal if state changes unexpectedly after delegation", func(t *testing.T) {
		if os.Getenv("BE_FATAL") == "1" {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			creatingState := regional.ResourceStateCreating
			resource := &regional.WorkspaceDomain{
				Status: regional.WorkspaceStatusDomain{
					StatusDomain: regional.StatusDomain{
						State: &creatingState,
					},
				},
			}

			mockPlugin := NewMockWorkspace(ctrl)
			mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, res *regional.WorkspaceDomain) error {
					// Plugin modifies the state
					activeState := regional.ResourceStateActive
					res.Status.State = &activeState
					return nil
				})

			handler := NewWorkspacePluginHandler(NewMockRepo[*regional.WorkspaceDomain](ctrl), mockPlugin)

			handler.HandleReconcile(context.Background(), resource)
			return
		}

		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=TestWorkspacePluginHandler_HandleReconcile/should_fatal_if_state_changes_unexpectedly_after_delegation")
		cmd.Env = append(os.Environ(), "BE_FATAL=1")
		err := cmd.Run()

		if e, ok := err.(*exec.ExitError); ok && !e.Success() { //nolint:errorlint // acceptable for tests
			// Test passed because the subprocess exited with a non-zero status code.
			return
		}
		t.Fatalf("process ran with err %v, want exit status 1", err)
	})
}
