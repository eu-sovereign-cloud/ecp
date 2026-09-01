package kubernetes_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"

	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
)

func TestSecurityGroupRulePluginHandler_HandleReconcile(t *testing.T) {
	var (
		errPlugin = errors.New("plugin failed")
		errRepo   = errors.New("repo error")
	)

	t.Run("should call plugin update and write no status when resource is active and in sync", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &securitygroupruledom.SecurityGroupRule{
			Status: &securitygroupruledom.SecurityGroupRuleStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateActive,
				},
			},
		}

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](ctrl)
		mockPlugin := NewMockSecurityGroupRulePlugin(ctrl)
		mockPlugin.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 0)

		err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
	})

	t.Run("should set state to creating and requeue when resource is pending", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &securitygroupruledom.SecurityGroupRule{
			Status: &securitygroupruledom.SecurityGroupRuleStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStatePending,
				},
			},
		}

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *securitygroupruledom.SecurityGroupRule) (*securitygroupruledom.SecurityGroupRule, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockSecurityGroupRulePlugin(ctrl)
		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 0)

		err := handler.HandleReconcile(context.Background(), resource)

		require.ErrorIs(t, err, backendport.StillProcessing)
	})

	t.Run("should call plugin create and set state to active when resource is creating", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &securitygroupruledom.SecurityGroupRule{
			Status: &securitygroupruledom.SecurityGroupRuleStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateCreating,
				},
			},
		}

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *securitygroupruledom.SecurityGroupRule) (*securitygroupruledom.SecurityGroupRule, error) {
				require.Equal(t, commondomain.ResourceStateActive, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockSecurityGroupRulePlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 0)

		err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
	})

	t.Run("should call plugin delete and set state to deleting when resource is deleting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		now := time.Now()
		resource := &securitygroupruledom.SecurityGroupRule{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &securitygroupruledom.SecurityGroupRuleStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateDeleting,
				},
			},
		}

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](ctrl)
		mockPlugin := NewMockSecurityGroupRulePlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(nil).Times(1)

		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 0)

		err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
	})

	t.Run("should set state to error and requeue when plugin create fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &securitygroupruledom.SecurityGroupRule{
			Status: &securitygroupruledom.SecurityGroupRuleStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateCreating,
				},
			},
		}

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *securitygroupruledom.SecurityGroupRule) (*securitygroupruledom.SecurityGroupRule, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockSecurityGroupRulePlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), resource).Return(errPlugin).Times(1)

		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 0)
		handler.MaxConditions = 1

		err := handler.HandleReconcile(context.Background(), resource)

		require.ErrorIs(t, err, backendport.StillProcessing)
	})

	t.Run("should return error when repo update fails after plugin failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &securitygroupruledom.SecurityGroupRule{
			Status: &securitygroupruledom.SecurityGroupRuleStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateCreating,
				},
			},
		}

		mockPlugin := NewMockSecurityGroupRulePlugin(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errPlugin)

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo)

		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 0)

		err := handler.HandleReconcile(context.Background(), resource)

		require.ErrorIs(t, err, errRepo)
	})

	t.Run("should set state to error and requeue when plugin delete fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		now := time.Now()
		resource := &securitygroupruledom.SecurityGroupRule{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					DeletedAt: &now,
				},
			},
			Status: &securitygroupruledom.SecurityGroupRuleStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateDeleting,
				},
			},
		}

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *securitygroupruledom.SecurityGroupRule) (*securitygroupruledom.SecurityGroupRule, error) {
				require.Equal(t, commondomain.ResourceStateError, res.Status.State)
				require.Len(t, res.Status.Conditions, 1)
				require.Equal(t, errPlugin.Error(), res.Status.Conditions[0].Message)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockSecurityGroupRulePlugin(ctrl)
		mockPlugin.EXPECT().Delete(gomock.Any(), resource).Return(errPlugin).Times(1)

		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 0)
		handler.MaxConditions = 1

		err := handler.HandleReconcile(context.Background(), resource)

		require.ErrorIs(t, err, backendport.StillProcessing)
	})

	t.Run("should set state to creating and requeue on retry create", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &securitygroupruledom.SecurityGroupRule{
			Status: &securitygroupruledom.SecurityGroupRuleStatus{
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

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, res *securitygroupruledom.SecurityGroupRule) (*securitygroupruledom.SecurityGroupRule, error) {
				require.Equal(t, commondomain.ResourceStateCreating, res.Status.State)
				return nil, nil
			}).Times(1)

		mockPlugin := NewMockSecurityGroupRulePlugin(ctrl)
		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 0)

		err := handler.HandleReconcile(context.Background(), resource)

		require.ErrorIs(t, err, backendport.StillProcessing)
	})

	t.Run("should do nothing for unhandled states", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &securitygroupruledom.SecurityGroupRule{
			Status: &securitygroupruledom.SecurityGroupRuleStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStateUpdating,
				},
			},
		}

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](ctrl)
		mockPlugin := NewMockSecurityGroupRulePlugin(ctrl)
		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 0)

		err := handler.HandleReconcile(context.Background(), resource)

		require.NoError(t, err)
	})

	t.Run("should return error when repo update fails in setResourceState", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resource := &securitygroupruledom.SecurityGroupRule{
			Status: &securitygroupruledom.SecurityGroupRuleStatus{
				Status: commondomain.Status{
					State: commondomain.ResourceStatePending,
				},
			},
		}

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](ctrl)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errRepo).Times(1)

		mockPlugin := NewMockSecurityGroupRulePlugin(ctrl)
		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 0)

		err := handler.HandleReconcile(context.Background(), resource)

		require.ErrorIs(t, err, errRepo)
	})

}
