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

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"

	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
)

func TestSubnetPluginHandler_HandleReconcile(t *testing.T) {
	var (
		errPlugin = errors.New("plugin failed")
		errRepo   = errors.New("repo error")
	)

	t.Run("should call plugin update and write no status when resource is active and in sync", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &subnetdom.Subnet{
			Status: &subnetdom.SubnetStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateActive,
				},
			},
		}

		mockRepo := NewMockRepo[*subnetdom.Subnet](ctrl)
		mockPlugin := NewMockSubnetPlugin(ctrl)
		mockPlugin.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		handler := NewSubnetPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should set state to creating and requeue when resource is pending", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &subnetdom.Subnet{
			Status: &subnetdom.SubnetStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStatePending,
				},
			},
		}

		mockRepo := NewMockRepo[*subnetdom.Subnet](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *subnetdom.Subnet) (*subnetdom.Subnet, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockSubnetPlugin(ctrl)
		handler := NewSubnetPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should call plugin create and set state to active when resource is creating", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &subnetdom.Subnet{
			Status: &subnetdom.SubnetStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateCreating,
				},
			},
		}

		mockRepo := NewMockRepo[*subnetdom.Subnet](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *subnetdom.Subnet) (*subnetdom.Subnet, error) {
				require.Equal(t, commondomain.ResourceStateActive, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockSubnetPlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewSubnetPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should call plugin delete and set state to deleting when resource is deleting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		now := time.Now()
		resource := &subnetdom.Subnet{
			RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{
						DeletedAt: &now,
					},
				},
			},
			Status: &subnetdom.SubnetStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateDeleting,
				},
			},
		}

		mockRepo := NewMockRepo[*subnetdom.Subnet](ctrl)
		mockPlugin := NewMockSubnetPlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewSubnetPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should set state to error and requeue when plugin create fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &subnetdom.Subnet{
			Status: &subnetdom.SubnetStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateCreating,
				},
			},
		}

		mockRepo := NewMockRepo[*subnetdom.Subnet](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *subnetdom.Subnet) (*subnetdom.Subnet, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockSubnetPlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(errPlugin).Times(1)

		handler := NewSubnetPluginHandler(mockRepo, mockPlugin, 0)
		handler.MaxConditions = 1

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should return error when repo update fails after plugin failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &subnetdom.Subnet{
			Status: &subnetdom.SubnetStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateCreating,
				},
			},
		}

		mockPlugin := NewMockSubnetPlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errPlugin)

		mockRepo := NewMockRepo[*subnetdom.Subnet](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo)

		handler := NewSubnetPluginHandler(mockRepo, mockPlugin, 0)

		_, err := handler.HandleReconcile(context.Background(), resource)

		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should set state to error and requeue when plugin delete fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		now := time.Now()
		resource := &subnetdom.Subnet{
			RegionalNetworkMetadata: commondomain.RegionalNetworkMetadata{
				RegionalMetadata: commondomain.RegionalMetadata{
					CommonMetadata: commondomain.CommonMetadata{
						DeletedAt: &now,
					},
				},
			},
			Status: &subnetdom.SubnetStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateDeleting,
				},
			},
		}

		mockRepo := NewMockRepo[*subnetdom.Subnet](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *subnetdom.Subnet) (*subnetdom.Subnet, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockSubnetPlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(errPlugin).Times(1)

		handler := NewSubnetPluginHandler(mockRepo, mockPlugin, 0)
		handler.MaxConditions = 1

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should set state to creating and requeue on retry create", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &subnetdom.Subnet{
			Status: &subnetdom.SubnetStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateError,
					Conditions: []commondomain.StatusCondition{
						{State: commondomain.ResourceStatePending, LastTransitionAt: time.Now().Add(-2 * time.Minute)},
						{State: commondomain.ResourceStateCreating, LastTransitionAt: time.Now().Add(-1 * time.Minute)},
						{State: commondomain.ResourceStateError, LastTransitionAt: time.Now()},
					},
				},
			},
		}

		mockRepo := NewMockRepo[*subnetdom.Subnet](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *subnetdom.Subnet) (*subnetdom.Subnet, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockSubnetPlugin(ctrl)
		handler := NewSubnetPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should do nothing for unhandled states", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &subnetdom.Subnet{
			Status: &subnetdom.SubnetStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateUpdating,
				},
			},
		}

		mockRepo := NewMockRepo[*subnetdom.Subnet](ctrl)
		mockPlugin := NewMockSubnetPlugin(ctrl)
		handler := NewSubnetPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should return error when repo update fails in setResourceState", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &subnetdom.Subnet{
			Status: &subnetdom.SubnetStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStatePending,
				},
			},
		}

		mockRepo := NewMockRepo[*subnetdom.Subnet](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo).Times(1)

		mockPlugin := NewMockSubnetPlugin(ctrl)
		handler := NewSubnetPluginHandler(mockRepo, mockPlugin, 0)

		_, err := handler.HandleReconcile(context.Background(), resource)

		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should fatal if state changes unexpectedly after delegation", func(t *testing.T) {
		if os.Getenv("BE_FATAL") == "1" {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resource := &subnetdom.Subnet{
				Status: &subnetdom.SubnetStatus{
					Status: commondomain.Status{
						State: commondomain.ResourceStateCreating,
					},
				},
			}

			mockPlugin := NewMockSubnetPlugin(ctrl)
			mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, res *subnetdom.Subnet) error {
					res.Status.State = commondomain.ResourceStateActive
					return nil
				})

			handler := NewSubnetPluginHandler(NewMockRepo[*subnetdom.Subnet](ctrl), mockPlugin, 0)
			handler.HandleReconcile(context.Background(), resource) //nolint:errcheck
			return
		}

		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=TestSubnetPluginHandler_HandleReconcile/should_fatal_if_state_changes_unexpectedly_after_delegation")
		cmd.Env = append(os.Environ(), "BE_FATAL=1")
		err := cmd.Run()

		if e, ok := errors.AsType[*exec.ExitError](err); ok && !e.Success() { //nolint:errorlint // acceptable for tests
			return
		}
		t.Fatalf("process ran with err %v, want exit status 1", err)
	})
}
