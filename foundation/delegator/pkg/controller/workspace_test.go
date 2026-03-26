package controller

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
	k8srt "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

func TestWorkspaceController_Reconcile(t *testing.T) {
	const (
		testName      = "test-ws"
		testNamespace = "default"
		testTenant    = "test-tenant"
	)

	var (
		errHandler   = errors.New("handler failed")
		pendingState = types.ResourceStatePending
	)

	// Common Setup
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(workspacev1.WorkspaceGVK.GroupVersion(), &workspacev1.Workspace{}, &workspacev1.WorkspaceList{})

	t.Run("should successfully reconcile a valid resource", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a valid K8s resource
		k8sRes := &workspacev1.Workspace{
			TypeMeta: metav1.TypeMeta{
				Kind:       workspacev1.WorkspaceGVK.Kind,
				APIVersion: workspacev1.WorkspaceGVK.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels: map[string]string{
					labels.InternalTenantLabel: testTenant,
				},
			},
			Spec: types.WorkspaceSpec{},
			Status: &types.WorkspaceStatus{
				State: &pendingState,
			},
		}

		//
		// And a Kubernetes which the schema contains the above resource
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(k8sRes).
			Build()

		//
		// And a repo that is expected to be called once
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockWorkspace(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(0)

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a workspace controller using these elements
		reconciler := NewWorkspaceController(
			fakeClient,
			mockRepo,
			mockPlugin,
			0,
			logger,
		)

		//
		// When we try to reconcile the resource
		req := k8srt.Request{NamespacedName: client.ObjectKey{Name: testName, Namespace: testNamespace}}
		res, err := (*GenericController[*regional.WorkspaceDomain])(reconciler).Reconcile(t.Context(), req)

		//
		// Then it should succeed
		require.NoError(t, err)
		require.Equal(t, k8srt.Result{}, res)

		//
		// And no error logs should be produced
		require.Empty(t, buf.String())
	})

	t.Run("should ignore when resource is not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given an empty K8s environment
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		//
		// And a repo that is not expected to be called
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockWorkspace(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(0)

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a workspace controller using these elements
		reconciler := NewWorkspaceController(
			fakeClient,
			mockRepo,
			mockPlugin,
			0,
			logger,
		)

		//
		// When we try to reconcile a missing resource
		req := k8srt.Request{NamespacedName: client.ObjectKey{Name: "missing", Namespace: testNamespace}}
		res, err := (*GenericController[*regional.WorkspaceDomain])(reconciler).Reconcile(t.Context(), req)

		//
		// Then it should return no error and no result
		require.NoError(t, err)
		require.Equal(t, k8srt.Result{}, res)

		//
		// And no error logs should be produced
		require.Empty(t, buf.String())
	})

	t.Run("should report an error when handler fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a valid K8s resource
		k8sRes := &workspacev1.Workspace{
			TypeMeta: metav1.TypeMeta{
				Kind:       workspacev1.WorkspaceGVK.Kind,
				APIVersion: workspacev1.WorkspaceGVK.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels: map[string]string{
					labels.InternalTenantLabel: testTenant,
				},
			},
			Spec: types.WorkspaceSpec{},
			Status: &types.WorkspaceStatus{
				State: &pendingState,
			},
		}

		//
		// And a Kubernetes which the schema contains the above resource
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(k8sRes).
			Build()

		//
		// And a repo which will return an error
		mockRepo := NewMockRepo[*regional.WorkspaceDomain](ctrl)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, errHandler).Times(1)

		//
		// And a plugin that is not expected to be called
		mockPlugin := NewMockWorkspace(ctrl)
		mockPlugin.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(0)

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a generic controller using these elements
		reconciler := NewWorkspaceController(
			fakeClient,
			mockRepo,
			mockPlugin,
			5*time.Minute,
			logger,
		)

		//
		// When we try to reconcile the resource
		req := k8srt.Request{NamespacedName: client.ObjectKey{Name: testName, Namespace: testNamespace}}
		res, err := (*GenericController[*regional.WorkspaceDomain])(reconciler).Reconcile(t.Context(), req)

		//
		// Then it should return the handler error
		require.ErrorIs(t, err, errHandler)

		//
		// And the error should be logged
		require.Contains(t, buf.String(), "handler failed to reconcile")

		//
		// And the result has the requeue properly set
		require.Equal(t, k8srt.Result{RequeueAfter: 5 * time.Minute}, res)
	})
}
