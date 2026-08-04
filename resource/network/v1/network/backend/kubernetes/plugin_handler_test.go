package kubernetes_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"

	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
)

func TestNetworkPluginHandler_HandleReconcile(t *testing.T) {
	var (
		errPlugin = errors.New("plugin failed")
		errRepo   = errors.New("repo error")
	)

	activeNetwork := func() *netdom.Network {
		return &netdom.Network{
			Status: &netdom.NetworkStatus{
				Status: commondomain.Status{
					State:      commondomain.ResourceStateActive,
					Conditions: []commondomain.StatusCondition{{Type: "Reconcile", State: commondomain.ResourceStateActive}},
				},
			},
		}
	}

	// An active resource is handed to the plugin's Update on every pass so a changed spec can
	// reach the provider. Nothing is written when the provider is already in sync - the
	// controller watches its own status writes, so a write per pass would never settle.
	t.Run("should call plugin update and write no status when resource is active and in sync", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockPlugin := NewMockNetworkPlugin(ctrl)
		mockPlugin.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), activeNetwork())

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should requeue without touching status while an update is still processing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockPlugin := NewMockNetworkPlugin(ctrl)
		mockPlugin.EXPECT().Update(gomock.Any(), gomock.Any()).Return(backendport.ErrStillProcessing).Times(1)
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), activeNetwork())

		require.NoError(t, err)
		require.True(t, requeue)
	})

	// A change the provider can never apply is reported and dropped. Retrying would re-issue a
	// request it has already refused; the resource stays active because it is still running.
	t.Run("should report an unsupported update on the resource and not requeue", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := activeNetwork()

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *netdom.Network) (*netdom.Network, error) {
				require.Equal(t, commondomain.ResourceStateActive, res.Status.State)
				require.Equal(t, "UpdateFailed", res.Status.Conditions[0].Type)
				require.Contains(t, res.Status.Conditions[0].Message, "region is immutable")
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockNetworkPlugin(ctrl)
		mockPlugin.EXPECT().Update(gomock.Any(), gomock.Any()).
			Return(fmt.Errorf("%w: region is immutable", backendport.ErrNotSupported)).Times(1)
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue, "an operation the provider refuses outright must not be retried")
	})

	// The controller reconciles on its own status writes, so re-reporting an unchanged failure
	// would loop forever. The second identical failure must write nothing.
	t.Run("should report a repeated identical update failure only once", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := activeNetwork()
		failure := fmt.Errorf("%w: region is immutable", backendport.ErrNotSupported)

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

		mockPlugin := NewMockNetworkPlugin(ctrl)
		mockPlugin.EXPECT().Update(gomock.Any(), gomock.Any()).Return(failure).Times(2)
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		_, err := handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)
		_, err = handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)

		require.Len(t, resource.Status.Conditions, 2, "the repeat must not append another condition")
	})

	// A transient failure is retried, unlike an unsupported one.
	t.Run("should requeue a transient update failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

		mockPlugin := NewMockNetworkPlugin(ctrl)
		mockPlugin.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errPlugin).Times(1)
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), activeNetwork())

		require.NoError(t, err)
		require.True(t, requeue)
	})

	// Once the provider accepts the change, the reported failure is retracted.
	t.Run("should clear a reported update failure once the update succeeds", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := activeNetwork()
		resource.Status.PushCondition(commonbackend.UpdateFailedCondition(commondomain.ResourceStateActive, "region is immutable"))

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *netdom.Network) (*netdom.Network, error) {
				require.Equal(t, "Reconcile", res.Status.Conditions[0].Type)
				require.Equal(t, commondomain.ResourceStateActive, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockNetworkPlugin(ctrl)
		mockPlugin.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	// A delete request on an active resource still takes the lifecycle path, not the update one.
	t.Run("should take the delete path when an active resource is marked for deletion", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		deletedAt := time.Now()
		resource := activeNetwork()
		resource.DeletedAt = &deletedAt

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *netdom.Network) (*netdom.Network, error) {
				require.Equal(t, commondomain.ResourceStateDeleting, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockNetworkPlugin(ctrl)
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should set state to creating and requeue when resource is pending", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &netdom.Network{
			Status: &netdom.NetworkStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStatePending,
				},
			},
		}

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *netdom.Network) (*netdom.Network, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockNetworkPlugin(ctrl)
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should call plugin create and set state to active when resource is creating", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &netdom.Network{
			Status: &netdom.NetworkStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateCreating,
				},
			},
		}

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *netdom.Network) (*netdom.Network, error) {
				require.Equal(t, commondomain.ResourceStateActive, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockNetworkPlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should call plugin delete and set state to deleting when resource is deleting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		now := time.Now()
		resource := &netdom.Network{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &netdom.NetworkStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateDeleting,
				},
			},
		}

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockPlugin := NewMockNetworkPlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should set state to error and requeue when plugin create fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &netdom.Network{
			Status: &netdom.NetworkStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateCreating,
				},
			},
		}

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *netdom.Network) (*netdom.Network, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockNetworkPlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(errPlugin).Times(1)

		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)
		handler.MaxConditions = 1

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should return error when repo update fails after plugin failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &netdom.Network{
			Status: &netdom.NetworkStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateCreating,
				},
			},
		}

		mockPlugin := NewMockNetworkPlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errPlugin)

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo)

		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		_, err := handler.HandleReconcile(context.Background(), resource)

		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should set state to error and requeue when plugin delete fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		now := time.Now()
		resource := &netdom.Network{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &netdom.NetworkStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateDeleting,
				},
			},
		}

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *netdom.Network) (*netdom.Network, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockNetworkPlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(errPlugin).Times(1)

		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)
		handler.MaxConditions = 1

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should set state to creating and requeue on retry create", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &netdom.Network{
			Status: &netdom.NetworkStatus{
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

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *netdom.Network) (*netdom.Network, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockNetworkPlugin(ctrl)
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should do nothing for unhandled states", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &netdom.Network{
			Status: &netdom.NetworkStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateUpdating,
				},
			},
		}

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockPlugin := NewMockNetworkPlugin(ctrl)
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should return error when repo update fails in setResourceState", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &netdom.Network{
			Status: &netdom.NetworkStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStatePending,
				},
			},
		}

		mockRepo := NewMockRepo[*netdom.Network](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo).Times(1)

		mockPlugin := NewMockNetworkPlugin(ctrl)
		handler := NewNetworkPluginHandler(mockRepo, mockPlugin, 0)

		_, err := handler.HandleReconcile(context.Background(), resource)

		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should fatal if state changes unexpectedly after delegation", func(t *testing.T) {
		if os.Getenv("BE_FATAL") == "1" {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resource := &netdom.Network{
				Status: &netdom.NetworkStatus{
					Status: commondomain.Status{
						State: commondomain.ResourceStateCreating,
					},
				},
			}

			mockPlugin := NewMockNetworkPlugin(ctrl)
			mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, res *netdom.Network) error {
					res.Status.State = commondomain.ResourceStateActive
					return nil
				})

			handler := NewNetworkPluginHandler(NewMockRepo[*netdom.Network](ctrl), mockPlugin, 0)
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
