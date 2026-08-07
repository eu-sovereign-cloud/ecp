package kubernetes_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	k8srt "sigs.k8s.io/controller-runtime/pkg/reconcile"

	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"

	. "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

func TestWorkspaceController_Reconcile(t *testing.T) {
	const (
		testName      = "test-ws"
		testNamespace = "test-namespace"
		testTenant    = "test-tenant"
	)

	errHandler := errors.New("handler error")

	newScheme := func() *runtime.Scheme {
		s := runtime.NewScheme()
		_ = AddToScheme(s)
		return s
	}

	newK8sResource := func() *Workspace {
		return &Workspace{
			TypeMeta: metav1.TypeMeta{
				Kind:       WorkspaceKind,
				APIVersion: GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:       testName,
				Namespace:  testNamespace,
				Finalizers: []string{"secapi.cloud.foundation/cleanup"},
				Labels: map[string]string{
					k8slabels.InternalTenantLabel: testTenant,
				},
			},
			Status: &WorkspaceStatus{State: schemav1.ResourceStatePending},
		}
	}

	req := k8srt.Request{NamespacedName: types.NamespacedName{Name: testName, Namespace: testNamespace}}

	t.Run("should successfully reconcile a valid resource", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockRepo := NewMockRepo[*wsdom.Workspace](mc)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

		mockPlugin := NewMockWorkspacePlugin(mc)

		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			WithObjects(newK8sResource()).
			Build()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 1)
		gc := frameworkcontroller.NewGenericController[*wsdom.Workspace](
			fakeClient,
			WorkspaceFromCR,
			handler,
			&Workspace{},
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

		mockRepo := NewMockRepo[*wsdom.Workspace](mc)
		mockPlugin := NewMockWorkspacePlugin(mc)

		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			Build()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 1)
		gc := frameworkcontroller.NewGenericController[*wsdom.Workspace](
			fakeClient,
			WorkspaceFromCR,
			handler,
			&Workspace{},
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

		mockRepo := NewMockRepo[*wsdom.Workspace](mc)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errHandler).Times(1)

		mockPlugin := NewMockWorkspacePlugin(mc)

		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			WithObjects(newK8sResource()).
			Build()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		requeueAfter := 5 * time.Minute
		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 1)
		gc := frameworkcontroller.NewGenericController[*wsdom.Workspace](
			fakeClient,
			WorkspaceFromCR,
			handler,
			&Workspace{},
			requeueAfter,
			logger,
			1,
		)

		res, err := gc.Reconcile(t.Context(), req)

		require.ErrorIs(t, err, errHandler)
		require.Contains(t, buf.String(), "handler failed to reconcile")
		require.Equal(t, k8srt.Result{RequeueAfter: requeueAfter}, res)
	})

	// The cleanup hook is the seam that lets a resource tear down the namespace it owns for its
	// children. It must run only after the plugin has finished deleting, and only while the
	// finalizer still holds the object — otherwise the side effect is lost.
	newDeletingResource := func() *Workspace {
		ws := newK8sResource()
		now := metav1.Now()
		ws.DeletionTimestamp = &now
		ws.Status = &WorkspaceStatus{State: schemav1.ResourceStateDeleting}
		return ws
	}

	newDeletingController := func(t *testing.T, fakeClient client.Client, requeueAfter time.Duration) *frameworkcontroller.GenericController[*wsdom.Workspace] {
		t.Helper()
		mc := gomock.NewController(t)
		t.Cleanup(mc.Finish)

		mockRepo := NewMockRepo[*wsdom.Workspace](mc)
		mockPlugin := NewMockWorkspacePlugin(mc)
		mockPlugin.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		handler := NewWorkspacePluginHandler(mockRepo, mockPlugin, 1)
		gc := frameworkcontroller.NewGenericController[*wsdom.Workspace](
			fakeClient,
			WorkspaceFromCR,
			handler,
			&Workspace{},
			requeueAfter,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			1,
		)
		return &gc
	}

	t.Run("should run the cleanup hook before dropping the finalizer", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			WithObjects(newDeletingResource()).
			Build()

		calls := 0
		gc := newDeletingController(t, fakeClient, 0)
		gc.WithCleanup(func(ctx context.Context, ws *wsdom.Workspace) error {
			calls++
			require.Equal(t, testName, ws.Name)

			// The finalizer must still be present while cleanup runs — that is what guarantees
			// a retry if it fails.
			var current Workspace
			require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, &current))
			require.Contains(t, current.GetFinalizers(), "secapi.cloud.foundation/cleanup")
			return nil
		})

		res, err := gc.Reconcile(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, k8srt.Result{}, res)
		require.Equal(t, 1, calls, "cleanup must run exactly once")

		// Dropping the last finalizer on an object with a deletionTimestamp releases it, so a
		// NotFound here is the proof the finalizer was removed after cleanup ran.
		var after Workspace
		require.True(t, kerrs.IsNotFound(fakeClient.Get(t.Context(), req.NamespacedName, &after)))
	})

	t.Run("should keep the finalizer and requeue when the cleanup hook fails", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			WithObjects(newDeletingResource()).
			Build()

		errCleanup := errors.New("namespace still has children")
		requeueAfter := 5 * time.Minute
		gc := newDeletingController(t, fakeClient, requeueAfter)
		gc.WithCleanup(func(context.Context, *wsdom.Workspace) error { return errCleanup })

		res, err := gc.Reconcile(t.Context(), req)
		require.ErrorIs(t, err, errCleanup)
		require.Equal(t, k8srt.Result{RequeueAfter: requeueAfter}, res)

		var after Workspace
		require.NoError(t, fakeClient.Get(t.Context(), req.NamespacedName, &after))
		require.Contains(t, after.GetFinalizers(), "secapi.cloud.foundation/cleanup",
			"a failed cleanup must not release the object, or the side effect is orphaned")
	})
}
