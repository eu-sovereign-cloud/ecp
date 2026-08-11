package kubernetes_test

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	k8srt "sigs.k8s.io/controller-runtime/pkg/reconcile"

	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"

	. "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
)

func TestSecurityGroupRuleController_Reconcile(t *testing.T) {
	const (
		testName      = "test-sgr"
		testNamespace = "test-namespace"
		testTenant    = "test-tenant"
		testWorkspace = "test-workspace"
	)

	errHandler := errors.New("handler error")

	newScheme := func() *runtime.Scheme {
		s := runtime.NewScheme()
		_ = AddToScheme(s)
		return s
	}

	newK8sResource := func() *SecurityGroupRule {
		return &SecurityGroupRule{
			TypeMeta: metav1.TypeMeta{
				Kind:       SecurityGroupRuleKind,
				APIVersion: GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:       testName,
				Namespace:  testNamespace,
				Finalizers: []string{"secapi.cloud.foundation/cleanup"},
				Labels: map[string]string{
					k8slabels.InternalTenantLabel:    testTenant,
					k8slabels.InternalWorkspaceLabel: testWorkspace,
				},
			},
			Spec:   SecurityGroupRuleSpec{Direction: SecurityGroupRuleSpecDirection("ingress")},
			Status: &SecurityGroupRuleStatus{State: schemav1.ResourceStatePending},
		}
	}

	req := k8srt.Request{NamespacedName: types.NamespacedName{Name: testName, Namespace: testNamespace}}

	t.Run("should successfully reconcile a valid resource", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](mc)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

		mockPlugin := NewMockSecurityGroupRulePlugin(mc)

		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			WithObjects(newK8sResource()).
			Build()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 1)
		gc := frameworkcontroller.NewGenericController[*securitygroupruledom.SecurityGroupRule](
			fakeClient,
			SecurityGroupRuleFromCR,
			handler,
			&SecurityGroupRule{},
			0,
			logger,
			1,
		)

		res, err := gc.Reconcile(t.Context(), req)

		require.NoError(t, err)
		require.Equal(t, k8srt.Result{}, res)
	})

	t.Run("should ignore when resource is not found", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](mc)
		mockPlugin := NewMockSecurityGroupRulePlugin(mc)

		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			Build()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 1)
		gc := frameworkcontroller.NewGenericController[*securitygroupruledom.SecurityGroupRule](
			fakeClient,
			SecurityGroupRuleFromCR,
			handler,
			&SecurityGroupRule{},
			0,
			logger,
			1,
		)

		res, err := gc.Reconcile(t.Context(), req)

		require.NoError(t, err)
		require.Equal(t, k8srt.Result{}, res)
	})

	t.Run("should report an error when handler fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockRepo := NewMockRepo[*securitygroupruledom.SecurityGroupRule](mc)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errHandler).Times(1)

		mockPlugin := NewMockSecurityGroupRulePlugin(mc)

		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			WithObjects(newK8sResource()).
			Build()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		requeueAfter := 5 * time.Minute
		handler := NewSecurityGroupRulePluginHandler(mockRepo, mockPlugin, 1)
		gc := frameworkcontroller.NewGenericController[*securitygroupruledom.SecurityGroupRule](
			fakeClient,
			SecurityGroupRuleFromCR,
			handler,
			&SecurityGroupRule{},
			requeueAfter,
			logger,
			1,
		)

		res, err := gc.Reconcile(t.Context(), req)

		require.ErrorIs(t, err, errHandler)
		require.Contains(t, buf.String(), "handler failed to reconcile")
		require.Equal(t, k8srt.Result{}, res,
			"a Result returned alongside an error is discarded by controller-runtime; the error alone drives the backoff requeue")
	})
}
