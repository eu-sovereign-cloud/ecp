package kubernetes_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role/v1"
	. "github.com/eu-sovereign-cloud/ecp/resource/authorization/role/v1/backend/kubernetes"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

func TestRolePluginHandler_HandleReconcile(t *testing.T) {
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
		resource := &roledom.Role{
			Status: &roledom.RoleStatus{
				Status: commondomain.Status{
					State: activeState,
				},
			},
		}

		//
		// And a repo and plugin that are not expected to be called
		mockRepo := NewMockRepo[*roledom.Role](ctrl)
		mockPlugin := NewMockRolePlugin(ctrl)

		//
		// And a role plugin handler
		handler := NewRolePluginHandler(mockRepo, mockPlugin, 0)

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
		resource := &roledom.Role{
			Status: &roledom.RoleStatus{
				Status: commondomain.Status{
					State: pendingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*roledom.Role](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *roledom.Role) (*roledom.Role, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockRolePlugin(ctrl)

		//
		// And a role plugin handler
		handler := NewRolePluginHandler(mockRepo, mockPlugin, 0)

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
		resource := &roledom.Role{
			Status: &roledom.RoleStatus{
				Status: commondomain.Status{
					State: creatingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state
		mockRepo := NewMockRepo[*roledom.Role](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *roledom.Role) (*roledom.Role, error) {
				require.Equal(t, commondomain.ResourceStateActive, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is expected to be called to create the resource
		mockPlugin := NewMockRolePlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a role plugin handler
		handler := NewRolePluginHandler(mockRepo, mockPlugin, 0)

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
		resource := &roledom.Role{
			GlobalTenantMetadata: commondomain.GlobalTenantMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &roledom.RoleStatus{
				Status: commondomain.Status{
					State: deletingState,
				},
			},
		}

		//
		// And a repo that is not expected to be called
		mockRepo := NewMockRepo[*roledom.Role](ctrl)

		//
		// And a plugin that is expected to be called to delete the resource
		mockPlugin := NewMockRolePlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a role plugin handler
		handler := NewRolePluginHandler(mockRepo, mockPlugin, 0)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed and not request a requeue
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should set resource to error state when plugin create fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with creating state
		creatingState := commondomain.ResourceStateCreating
		resource := &roledom.Role{
			Status: &roledom.RoleStatus{
				Status: commondomain.Status{
					State: creatingState,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to error
		mockRepo := NewMockRepo[*roledom.Role](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *roledom.Role) (*roledom.Role, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that will fail
		mockPlugin := NewMockRolePlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And a role plugin handler
		handler := NewRolePluginHandler(mockRepo, mockPlugin, 0)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should succeed (error was persisted) and request a requeue
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should return error when repo update fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with pending state
		pendingState := commondomain.ResourceStatePending
		resource := &roledom.Role{
			Status: &roledom.RoleStatus{
				Status: commondomain.Status{
					State: pendingState,
				},
			},
		}

		//
		// And a repo that will fail
		mockRepo := NewMockRepo[*roledom.Role](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockRolePlugin(ctrl)

		//
		// And a role plugin handler
		handler := NewRolePluginHandler(mockRepo, mockPlugin, 0)

		//
		// When we reconcile the resource
		requeue, err := handler.HandleReconcile(context.Background(), resource)

		//
		// Then it should return an error and request a requeue
		require.ErrorIs(t, err, errRepo)
		require.True(t, requeue)
	})
}
