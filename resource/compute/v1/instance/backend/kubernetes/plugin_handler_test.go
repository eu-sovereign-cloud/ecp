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

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"

	. "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
)

const restartIDA = "restart-A"

func activeInstance(powerState instancedom.PowerState) *instancedom.Instance {
	return &instancedom.Instance{
		Status: &instancedom.InstanceStatus{
			Status:     commondomain.Status{State: commondomain.ResourceStateActive},
			PowerState: powerState,
		},
	}
}

func TestInstancePluginHandler_PowerReconcile(t *testing.T) {
	t.Run("start: powers on when desired=on and currently off", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := activeInstance(instancedom.PowerStateOff)
		resource.DesiredPowerState = instancedom.PowerStateOn

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Equal(t, instancedom.PowerStateOn, res.Status.PowerState)
				require.NotNil(t, res.Status.PowerStateSince)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().PowerOn(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		requeue, err := handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("stop: powers off when desired=off and currently on", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := activeInstance(instancedom.PowerStateOn)
		resource.DesiredPowerState = instancedom.PowerStateOff

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Equal(t, instancedom.PowerStateOff, res.Status.PowerState)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().PowerOff(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		requeue, err := handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("no-op: no power action when desired matches current state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := activeInstance(instancedom.PowerStateOn)
		resource.DesiredPowerState = instancedom.PowerStateOn

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockPlugin := NewMockInstancePlugin(ctrl)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		requeue, err := handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("restart step 1: powers off and keeps the nonce when currently on", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := activeInstance(instancedom.PowerStateOn)
		resource.DesiredPowerState = instancedom.PowerStateOn
		resource.RestartID = restartIDA
		resource.RestartPhase = instancedom.RestartPhasePowerOff

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Equal(t, instancedom.PowerStateOff, res.Status.PowerState)
				return nil, nil
			}).Times(1)
		// Phase advance is compare-and-swap: load current (same id), then Update to power-on.
		mockRepo.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, m **instancedom.Instance) error {
				(*m).RestartID = restartIDA
				(*m).RestartPhase = instancedom.RestartPhasePowerOff
				return nil
			}).Times(1)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Equal(t, instancedom.RestartPhasePowerOn, res.RestartPhase)
				require.Equal(t, restartIDA, res.RestartID, "id must persist across the cycle")
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().PowerOff(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		requeue, err := handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)
		require.True(t, requeue, "restart cycle should requeue to complete")
	})

	t.Run("restart power-on phase: powers on and clears restart when id still matches", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := activeInstance(instancedom.PowerStateOff)
		resource.DesiredPowerState = instancedom.PowerStateOn
		resource.RestartID = restartIDA
		resource.RestartPhase = instancedom.RestartPhasePowerOn

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Equal(t, instancedom.PowerStateOn, res.Status.PowerState)
				return nil, nil
			}).Times(1)
		// Cleanup loads the current object (same id) and clears both restart annotations.
		mockRepo.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, m **instancedom.Instance) error {
				(*m).RestartID = restartIDA
				(*m).RestartPhase = instancedom.RestartPhasePowerOn
				return nil
			}).Times(1)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Empty(t, res.RestartID, "id must be cleared once the cycle completes")
				require.Empty(t, res.RestartPhase, "phase must be cleared once the cycle completes")
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().PowerOn(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		requeue, err := handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("restart power-on phase: does not clear a superseding restart", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := activeInstance(instancedom.PowerStateOff)
		resource.RestartID = restartIDA
		resource.RestartPhase = instancedom.RestartPhasePowerOn

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		// A newer restart (restart-B) is now persisted; cleanup must not clobber it (no Update).
		mockRepo.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, m **instancedom.Instance) error {
				(*m).RestartID = "restart-B"
				(*m).RestartPhase = instancedom.RestartPhasePowerOff
				return nil
			}).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().PowerOn(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		requeue, err := handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("phase advance does not clobber a superseding restart", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := activeInstance(instancedom.PowerStateOn)
		resource.DesiredPowerState = instancedom.PowerStateOn
		resource.RestartID = restartIDA
		resource.RestartPhase = instancedom.RestartPhasePowerOff

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		// A newer restart (restart-B) arrived during power-off; the advance must not overwrite it.
		mockRepo.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, m **instancedom.Instance) error {
				(*m).RestartID = "restart-B"
				(*m).RestartPhase = instancedom.RestartPhasePowerOff
				return nil
			}).Times(1)
		// No Update: the older reconcile leaves restart-B for its own reconcile.

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().PowerOff(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		requeue, err := handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("restart phase takes precedence over a concurrent desired power state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// A stop (desired=off) arrived mid-restart; the power-on phase must still power ON.
		resource := activeInstance(instancedom.PowerStateOff)
		resource.DesiredPowerState = instancedom.PowerStateOff
		resource.RestartID = restartIDA
		resource.RestartPhase = instancedom.RestartPhasePowerOn

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Equal(t, instancedom.PowerStateOn, res.Status.PowerState)
				return nil, nil
			}).Times(1)
		mockRepo.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, m **instancedom.Instance) error {
				(*m).RestartID = restartIDA
				(*m).RestartPhase = instancedom.RestartPhasePowerOn
				return nil
			}).Times(1)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		// PowerOn (not PowerOff), proving the phase wins over desired=off.
		mockPlugin.EXPECT().PowerOn(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		requeue, err := handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("status write failure after provider success requeues with the error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		errStatus := errors.New("status write failed")
		resource := activeInstance(instancedom.PowerStateOff)
		resource.DesiredPowerState = instancedom.PowerStateOn

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errStatus).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().PowerOn(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		requeue, err := handler.HandleReconcile(context.Background(), resource)
		require.ErrorIs(t, err, errStatus)
		require.True(t, requeue)
	})

	t.Run("terminal provider error is recorded as a condition and requeued", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		errProvider := errors.New("provider power-on failed")
		resource := activeInstance(instancedom.PowerStateOff)
		resource.DesiredPowerState = instancedom.PowerStateOn

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Equal(t, commondomain.ResourceStateActive, res.Status.State,
					"lifecycle state must be preserved")
				require.Equal(t, "PowerManagementError", res.Status.Conditions[0].Type)
				require.Equal(t, "PowerOperationFailed", res.Status.Conditions[0].Reason)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().PowerOn(gomock.Any(), resource).Return(errProvider).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		requeue, err := handler.HandleReconcile(context.Background(), resource)
		require.ErrorIs(t, err, errProvider)
		require.True(t, requeue)
	})

	t.Run("preserves PowerStateSince when re-persisting an already-recorded state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Re-running the power-on phase when the instance is already on (e.g. cleanup previously
		// failed) must not rewrite the transition timestamp to the retry time.
		transition := time.Now().Add(-1 * time.Hour)
		resource := activeInstance(instancedom.PowerStateOn)
		resource.Status.PowerStateSince = &transition
		resource.RestartID = restartIDA
		resource.RestartPhase = instancedom.RestartPhasePowerOn

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.NotNil(t, res.Status.PowerStateSince)
				require.True(t, res.Status.PowerStateSince.Equal(transition),
					"timestamp must be preserved when the state is unchanged")
				return nil, nil
			}).Times(1)
		mockRepo.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, m **instancedom.Instance) error {
				(*m).RestartID = restartIDA
				(*m).RestartPhase = instancedom.RestartPhasePowerOn
				return nil
			}).Times(1)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().PowerOn(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		_, err := handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)
	})

	malformed := []struct {
		name  string
		apply func(*instancedom.Instance)
	}{
		{"restart-id without phase", func(i *instancedom.Instance) { i.RestartID = "x" }},
		{"phase without restart-id", func(i *instancedom.Instance) { i.RestartPhase = instancedom.RestartPhasePowerOn }},
		{"unknown phase value", func(i *instancedom.Instance) { i.RestartID = "x"; i.RestartPhase = "reboot" }},
		{"invalid desired power state", func(i *instancedom.Instance) { i.DesiredPowerState = "paused" }},
	}
	for _, tc := range malformed {
		t.Run("malformed intent surfaces an error condition and requeues: "+tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resource := activeInstance(instancedom.PowerStateOn)
			tc.apply(resource)

			mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
			mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
					require.Equal(t, commondomain.ResourceStateActive, res.Status.State,
						"lifecycle state must be preserved")
					require.Equal(t, "PowerManagementError", res.Status.Conditions[0].Type)
					return nil, nil
				}).Times(1)

			// No plugin calls for malformed intent.
			mockPlugin := NewMockInstancePlugin(ctrl)

			handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
			requeue, err := handler.HandleReconcile(context.Background(), resource)
			require.NoError(t, err)
			require.True(t, requeue)
		})
	}

	t.Run("power op still processing requeues without a status change", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := activeInstance(instancedom.PowerStateOff)
		resource.DesiredPowerState = instancedom.PowerStateOn

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().PowerOn(gomock.Any(), resource).Return(backendport.ErrStillProcessing).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		requeue, err := handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("power management is skipped while the instance is not active", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &instancedom.Instance{
			Status: &instancedom.InstanceStatus{
				Status:     commondomain.Status{State: commondomain.ResourceStateCreating},
				PowerState: instancedom.PowerStateOff,
			},
			DesiredPowerState: instancedom.PowerStateOn,
		}

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		// Creating state → falls through to lifecycle (create), no power op invoked.
		_, err := handler.HandleReconcile(context.Background(), resource)
		require.NoError(t, err)
	})
}

func TestInstancePluginHandler_HandleReconcile(t *testing.T) {
	var (
		errPlugin = errors.New("plugin failed")
		errRepo   = errors.New("repo error")
	)

	t.Run("should do nothing if resource is active", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &instancedom.Instance{
			Status: &instancedom.InstanceStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateActive,
				},
			},
		}

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockPlugin := NewMockInstancePlugin(ctrl)
		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should set state to creating and requeue when resource is pending", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &instancedom.Instance{
			Status: &instancedom.InstanceStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStatePending,
				},
			},
		}

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should call plugin create and set state to active when resource is creating", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &instancedom.Instance{
			Status: &instancedom.InstanceStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateCreating,
				},
			},
		}

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Equal(t, commondomain.ResourceStateActive, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should call plugin delete and set state to deleting when resource is deleting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		now := time.Now()
		resource := &instancedom.Instance{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &instancedom.InstanceStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateDeleting,
				},
			},
		}

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should set state to error and requeue when plugin create fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &instancedom.Instance{
			Status: &instancedom.InstanceStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateCreating,
				},
			},
		}

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(errPlugin).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		handler.MaxConditions = 1

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should return error when repo update fails after plugin failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &instancedom.Instance{
			Status: &instancedom.InstanceStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateCreating,
				},
			},
		}

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errPlugin)

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)

		_, err := handler.HandleReconcile(context.Background(), resource)

		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should set state to error and requeue when plugin delete fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		now := time.Now()
		resource := &instancedom.Instance{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &instancedom.InstanceStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateDeleting,
				},
			},
		}

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(errPlugin).Times(1)

		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)
		handler.MaxConditions = 1

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should set state to creating and requeue on retry create", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &instancedom.Instance{
			Status: &instancedom.InstanceStatus{
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

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *instancedom.Instance) (*instancedom.Instance, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("should do nothing for unhandled states", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &instancedom.Instance{
			Status: &instancedom.InstanceStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateUpdating,
				},
			},
		}

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockPlugin := NewMockInstancePlugin(ctrl)
		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)

		requeue, err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("should return error when repo update fails in setResourceState", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &instancedom.Instance{
			Status: &instancedom.InstanceStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStatePending,
				},
			},
		}

		mockRepo := NewMockRepo[*instancedom.Instance](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo).Times(1)

		mockPlugin := NewMockInstancePlugin(ctrl)
		handler := NewInstancePluginHandler(mockRepo, mockPlugin, 0)

		_, err := handler.HandleReconcile(context.Background(), resource)

		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should fatal if state changes unexpectedly after delegation", func(t *testing.T) {
		if os.Getenv("BE_FATAL") == "1" {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resource := &instancedom.Instance{
				Status: &instancedom.InstanceStatus{
					Status: commondomain.Status{
						State: commondomain.ResourceStateCreating,
					},
				},
			}

			mockPlugin := NewMockInstancePlugin(ctrl)
			mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, res *instancedom.Instance) error {
					res.Status.State = commondomain.ResourceStateActive
					return nil
				})

			handler := NewInstancePluginHandler(NewMockRepo[*instancedom.Instance](ctrl), mockPlugin, 0)
			handler.HandleReconcile(context.Background(), resource) //nolint:errcheck
			return
		}

		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=TestInstancePluginHandler_HandleReconcile/should_fatal_if_state_changes_unexpectedly_after_delegation")
		cmd.Env = append(os.Environ(), "BE_FATAL=1")
		err := cmd.Run()

		if e, ok := errors.AsType[*exec.ExitError](err); ok && !e.Success() { //nolint:errorlint // acceptable for tests
			return
		}
		t.Fatalf("process ran with err %v, want exit status 1", err)
	})
}
