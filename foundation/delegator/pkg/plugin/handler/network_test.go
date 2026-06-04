package handler

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

func TestNetworkPluginHandler_HandleReconcile(t *testing.T) {
	var (
		errPlugin = errors.New("plugin failed")
		errRepo   = errors.New("repo error")
	)

	t.Run("should do nothing if resource is active", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with active state
		resource := &regional.NetworkDomain{
			Status: &regional.NetworkStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: regional.ResourceStateActive,
				},
			},
		}

		//
		// And a repo and plugin that are not expected to be called
		mockRepo := NewMockRepo[*regional.NetworkDomain](ctrl)
		mockPlugin := NewMockNetwork(ctrl)

		//
		// And a network plugin handler
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin)

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
		resource := &regional.NetworkDomain{
			Status: &regional.NetworkStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: regional.ResourceStatePending,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to creating
		mockRepo := NewMockRepo[*regional.NetworkDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.NetworkDomain) (*regional.NetworkDomain, error) {
				require.Equal(t, regional.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockNetwork(ctrl)

		//
		// And a network plugin handler
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin)

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
		resource := &regional.NetworkDomain{
			Status: &regional.NetworkStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: regional.ResourceStateCreating,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to active
		mockRepo := NewMockRepo[*regional.NetworkDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.NetworkDomain) (*regional.NetworkDomain, error) {
				require.Equal(t, regional.ResourceStateActive, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is expected to be called to create the resource
		mockPlugin := NewMockNetwork(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a network plugin handler
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin)

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
		now := time.Now()
		resource := &regional.NetworkDomain{
			Metadata: regional.Metadata{
				CommonMetadata: model.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &regional.NetworkStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: regional.ResourceStateDeleting,
				},
			},
		}

		//
		// And a repo that is not expected to be called
		mockRepo := NewMockRepo[*regional.NetworkDomain](ctrl)

		//
		// And a plugin that is expected to be called to delete the resource
		mockPlugin := NewMockNetwork(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(nil).Times(1)

		//
		// And a network plugin handler
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin)

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
		resource := &regional.NetworkDomain{
			Status: &regional.NetworkStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: regional.ResourceStateCreating,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to error
		mockRepo := NewMockRepo[*regional.NetworkDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.NetworkDomain) (*regional.NetworkDomain, error) {
				require.Equal(t, regional.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that returns an error on create
		mockPlugin := NewMockNetwork(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And a network plugin handler
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin)
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
		resource := &regional.NetworkDomain{
			Status: &regional.NetworkStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: regional.ResourceStateCreating,
				},
			},
		}

		//
		// And a plugin that returns an error
		mockPlugin := NewMockNetwork(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errPlugin)

		//
		// And a repo that returns an error on update
		mockRepo := NewMockRepo[*regional.NetworkDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo)

		//
		// And a network plugin handler
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin)

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
		now := time.Now()
		resource := &regional.NetworkDomain{
			Metadata: regional.Metadata{
				CommonMetadata: model.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &regional.NetworkStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: regional.ResourceStateDeleting,
				},
			},
		}

		//
		// And a repo that is expected to be called once to update state to error
		mockRepo := NewMockRepo[*regional.NetworkDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.NetworkDomain) (*regional.NetworkDomain, error) {
				require.Equal(t, regional.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that returns an error on delete
		mockPlugin := NewMockNetwork(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(errPlugin).Times(1)

		//
		// And a network plugin handler
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin)
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
		resource := &regional.NetworkDomain{
			Status: &regional.NetworkStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: regional.ResourceStateError,
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
		mockRepo := NewMockRepo[*regional.NetworkDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *regional.NetworkDomain) (*regional.NetworkDomain, error) {
				require.Equal(t, regional.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockNetwork(ctrl)

		//
		// And a network plugin handler
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin)

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
		resource := &regional.NetworkDomain{
			Status: &regional.NetworkStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: regional.ResourceStateUpdating,
				},
			},
		}

		//
		// And a repo and plugin that are not expected to be called
		mockRepo := NewMockRepo[*regional.NetworkDomain](ctrl)
		mockPlugin := NewMockNetwork(ctrl)

		//
		// And a network plugin handler
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin)

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
		resource := &regional.NetworkDomain{
			Status: &regional.NetworkStatusDomain{
				StatusDomain: regional.StatusDomain{
					State: regional.ResourceStatePending,
				},
			},
		}

		//
		// And a repo that returns an error on update
		mockRepo := NewMockRepo[*regional.NetworkDomain](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockNetwork(ctrl)

		//
		// And a network plugin handler
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin)

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

			resource := &regional.NetworkDomain{
				Status: &regional.NetworkStatusDomain{
					StatusDomain: regional.StatusDomain{
						State: regional.ResourceStateCreating,
					},
				},
			}

			mockPlugin := NewMockNetwork(ctrl)
			mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, res *regional.NetworkDomain) error {
					res.Status.State = regional.ResourceStateActive
					return nil
				})

			handler := NewNetworkPluginHandler(NewMockRepo[*regional.NetworkDomain](ctrl), mockPlugin)
			handler.HandleReconcile(context.Background(), resource) //nolint:errcheck
			return
		}

		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=TestNetworkPluginHandler_HandleReconcile/should_fatal_if_state_changes_unexpectedly_after_delegation")
		cmd.Env = append(os.Environ(), "BE_FATAL=1")
		err := cmd.Run()

		if e, ok := errors.AsType[*exec.ExitError](err); ok && !e.Success() { //nolint:errorlint // acceptable for tests
			return
		}
		t.Fatalf("process ran with err %v, want exit status 1", err)
	})
}
