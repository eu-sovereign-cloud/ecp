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

	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	. "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment/backend/kubernetes"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

func TestRoleAssignmentPluginHandler_HandleReconcile(t *testing.T) {
	var (
		errPlugin = errors.New("plugin failed")
		errRepo   = errors.New("repo failed")
	)

	t.Run("should do nothing if resource is active and requires no changes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with active state
		activeState := commondomain.ResourceStateActive
		resource := &radom.RoleAssignment{
			Spec: radom.RoleAssignmentSpec{Roles: []string{"workspace-viewer"}},
			Status: &radom.RoleAssignmentStatus{
				Status: commondomain.Status{
					State: activeState,
				},
			},
		}

		//
		// And a repo and plugin that are not expected to be called
		mockRepo := NewMockRepo[*radom.RoleAssignment](ctrl)
		mockPlugin := NewMockRoleAssignmentPlugin(ctrl)

		//
		// And a role assignment plugin handler
		handler := NewRoleAssignmentPluginHandler(mockRepo, mockPlugin, 0)

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
		resource := &radom.RoleAssignment{
			Status: &radom.RoleAssignmentStatus{
				Status: commondomain.Status{
					State: pendingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*radom.RoleAssignment](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *radom.RoleAssignment) (*radom.RoleAssignment, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockRoleAssignmentPlugin(ctrl)

		//
		// And a role assignment plugin handler
		handler := NewRoleAssignmentPluginHandler(mockRepo, mockPlugin, 0)

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
		resource := &radom.RoleAssignment{
			Spec: radom.RoleAssignmentSpec{Roles: []string{"workspace-viewer"}},
			Status: &radom.RoleAssignmentStatus{
				Status: commondomain.Status{
					State: creatingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*radom.RoleAssignment](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *radom.RoleAssignment) (*radom.RoleAssignment, error) {
				require.Equal(t, commondomain.ResourceStateActive, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is expected to be called to create the resource
		mockPlugin := NewMockRoleAssignmentPlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a role assignment plugin handler
		handler := NewRoleAssignmentPluginHandler(mockRepo, mockPlugin, 0)

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
		resource := &radom.RoleAssignment{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &radom.RoleAssignmentStatus{
				Status: commondomain.Status{
					State: deletingState,
				},
			},
		}

		//
		// And a repo that is not expected to be called
		mockRepo := NewMockRepo[*radom.RoleAssignment](ctrl)

		//
		// And a plugin that is expected to be called to delete the resource
		mockPlugin := NewMockRoleAssignmentPlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a role assignment plugin handler
		handler := NewRoleAssignmentPluginHandler(mockRepo, mockPlugin, 0)

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
		errorState := commondomain.ResourceStateError
		resource := &radom.RoleAssignment{
			Status: &radom.RoleAssignmentStatus{
				Status: commondomain.Status{
					State: errorState,
					Conditions: []commondomain.StatusCondition{
						{State: commondomain.ResourceStateCreating},
						{State: commondomain.ResourceStateError},
					},
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to creating
		mockRepo := NewMockRepo[*radom.RoleAssignment](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *radom.RoleAssignment) (*radom.RoleAssignment, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockRoleAssignmentPlugin(ctrl)

		//
		// And a role assignment plugin handler
		handler := NewRoleAssignmentPluginHandler(mockRepo, mockPlugin, 0)

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
		creatingState := commondomain.ResourceStateCreating
		resource := &radom.RoleAssignment{
			Status: &radom.RoleAssignmentStatus{
				Status: commondomain.Status{
					State: creatingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to error
		mockRepo := NewMockRepo[*radom.RoleAssignment](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *radom.RoleAssignment) (*radom.RoleAssignment, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that returns an error on create
		mockPlugin := NewMockRoleAssignmentPlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And a role assignment plugin handler
		handler := NewRoleAssignmentPluginHandler(mockRepo, mockPlugin, 0)
		handler.MaxConditions = 1

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
		creatingState := commondomain.ResourceStateCreating
		resource := &radom.RoleAssignment{
			Status: &radom.RoleAssignmentStatus{
				Status: commondomain.Status{
					State: creatingState,
				},
			},
		}

		//
		// And a plugin that returns an error
		mockPlugin := NewMockRoleAssignmentPlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errPlugin)

		//
		// And a repo that returns an error on update
		mockRepo := NewMockRepo[*radom.RoleAssignment](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo)

		//
		// And a role assignment plugin handler
		handler := NewRoleAssignmentPluginHandler(mockRepo, mockPlugin, 0)

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
		pendingState := commondomain.ResourceStatePending
		resource := &radom.RoleAssignment{
			Status: &radom.RoleAssignmentStatus{
				Status: commondomain.Status{
					State: pendingState,
				},
			},
		}

		//
		// And a repo that returns an error on update
		mockRepo := NewMockRepo[*radom.RoleAssignment](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockRoleAssignmentPlugin(ctrl)

		//
		// And a role assignment plugin handler
		handler := NewRoleAssignmentPluginHandler(mockRepo, mockPlugin, 0)

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
		resource := &radom.RoleAssignment{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &radom.RoleAssignmentStatus{
				Status: commondomain.Status{
					State: deletingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to error
		mockRepo := NewMockRepo[*radom.RoleAssignment](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *radom.RoleAssignment) (*radom.RoleAssignment, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that returns an error on delete
		mockPlugin := NewMockRoleAssignmentPlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And a role assignment plugin handler
		handler := NewRoleAssignmentPluginHandler(mockRepo, mockPlugin, 0)
		handler.MaxConditions = 1

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should handle the error gracefully and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should fatal if state changes unexpectedly after delegation", func(t *testing.T) {
		if os.Getenv("BE_FATAL") == "1" {
			//
			// Given a controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			//
			// And a resource with creating state
			creatingState := commondomain.ResourceStateCreating
			resource := &radom.RoleAssignment{
				Status: &radom.RoleAssignmentStatus{
					Status: commondomain.Status{
						State: creatingState,
					},
				},
			}

			//
			// And a plugin that modifies the resource state during delegation
			mockPlugin := NewMockRoleAssignmentPlugin(ctrl)
			mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, res *radom.RoleAssignment) error {
					res.Status.State = commondomain.ResourceStateActive
					return nil
				})

			//
			// And a role assignment plugin handler
			handler := NewRoleAssignmentPluginHandler(NewMockRepo[*radom.RoleAssignment](ctrl), mockPlugin, 0)

			//
			// When we reconcile the resource
			handler.HandleReconcile(context.Background(), resource)

			//
			// Then the process should exit with a fatal error
			return
		}

		//
		// Given a command to run the test in a separate process
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=TestRoleAssignmentPluginHandler_HandleReconcile/should_fatal_if_state_changes_unexpectedly_after_delegation")
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
}
