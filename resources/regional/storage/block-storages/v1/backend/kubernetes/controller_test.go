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

	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/controller"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/labels"
	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"

	. "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/backend/kubernetes"
)

func TestBlockStorageController_Reconcile(t *testing.T) {
	const (
		testName      = "test-bs"
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

	newK8sResource := func() *BlockStorage {
		return &BlockStorage{
			TypeMeta: metav1.TypeMeta{
				Kind:       BlockStorageKind,
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
			Spec:   BlockStorageSpec{SizeGB: 10},
			Status: &BlockStorageStatus{State: schemav1.ResourceStatePending},
		}
	}

	req := k8srt.Request{NamespacedName: types.NamespacedName{Name: testName, Namespace: testNamespace}}

	t.Run("should successfully reconcile a valid resource", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockRepo := NewMockRepo[*bsdom.BlockStorage](mc)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

		mockPlugin := NewMockBlockStoragePlugin(mc)

		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			WithObjects(newK8sResource()).
			Build()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin, 1)
		gc := frameworkcontroller.NewGenericController[*bsdom.BlockStorage](
			fakeClient,
			MapCRToBlockStorageDomain,
			handler,
			&BlockStorage{},
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

		mockRepo := NewMockRepo[*bsdom.BlockStorage](mc)
		mockPlugin := NewMockBlockStoragePlugin(mc)

		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			Build()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin, 1)
		gc := frameworkcontroller.NewGenericController[*bsdom.BlockStorage](
			fakeClient,
			MapCRToBlockStorageDomain,
			handler,
			&BlockStorage{},
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

		mockRepo := NewMockRepo[*bsdom.BlockStorage](mc)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errHandler).Times(1)

		mockPlugin := NewMockBlockStoragePlugin(mc)

		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			WithObjects(newK8sResource()).
			Build()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		requeueAfter := 5 * time.Minute
		handler := NewBlockStoragePluginHandler(mockRepo, mockPlugin, 1)
		gc := frameworkcontroller.NewGenericController[*bsdom.BlockStorage](
			fakeClient,
			MapCRToBlockStorageDomain,
			handler,
			&BlockStorage{},
			requeueAfter,
			logger,
			1,
		)

		res, err := gc.Reconcile(t.Context(), req)

		require.ErrorIs(t, err, errHandler)
		require.Contains(t, buf.String(), "handler failed to reconcile")
		require.Equal(t, k8srt.Result{RequeueAfter: requeueAfter}, res)
	})
}
