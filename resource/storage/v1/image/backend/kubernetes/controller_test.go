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
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"

	. "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
)

func TestImageController_Reconcile(t *testing.T) {
	const (
		testName      = "test-img"
		testNamespace = "test-namespace"
		testTenant    = "test-tenant"
	)

	errHandler := errors.New("handler error")

	newScheme := func() *runtime.Scheme {
		s := runtime.NewScheme()
		_ = AddToScheme(s)
		return s
	}

	newK8sResource := func() *Image {
		return &Image{
			TypeMeta: metav1.TypeMeta{
				Kind:       ImageKind,
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
			Spec:   ImageSpec{CpuArchitecture: ImageSpecCpuArchitectureAmd64},
			Status: &ImageStatus{State: schemav1.ResourceStatePending},
		}
	}

	req := k8srt.Request{NamespacedName: types.NamespacedName{Name: testName, Namespace: testNamespace}}

	t.Run("should successfully reconcile a valid resource", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockRepo := NewMockRepo[*imgdom.Image](mc)
		mockDeps := NewMockDependencyResolver(mc)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		mockDeps.EXPECT().State(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, commondomain.ResourceStateActive, nil).Times(1)

		mockPlugin := NewMockImagePlugin(mc)

		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			WithObjects(newK8sResource()).
			Build()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		handler := NewImagePluginHandler(mockRepo, mockPlugin, 1, mockDeps)
		gc := frameworkcontroller.NewGenericController[*imgdom.Image](
			fakeClient,
			ImageFromCR,
			handler,
			&Image{},
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

		mockRepo := NewMockRepo[*imgdom.Image](mc)
		mockDeps := NewMockDependencyResolver(mc)
		mockPlugin := NewMockImagePlugin(mc)

		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			Build()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		handler := NewImagePluginHandler(mockRepo, mockPlugin, 1, mockDeps)
		gc := frameworkcontroller.NewGenericController[*imgdom.Image](
			fakeClient,
			ImageFromCR,
			handler,
			&Image{},
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

		mockRepo := NewMockRepo[*imgdom.Image](mc)
		mockDeps := NewMockDependencyResolver(mc)
		mockRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil, errHandler).Times(1)
		mockDeps.EXPECT().State(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, commondomain.ResourceStateActive, nil).Times(1)

		mockPlugin := NewMockImagePlugin(mc)

		fakeClient := fake.NewClientBuilder().
			WithScheme(newScheme()).
			WithObjects(newK8sResource()).
			Build()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		requeueAfter := 5 * time.Minute
		handler := NewImagePluginHandler(mockRepo, mockPlugin, 1, mockDeps)
		gc := frameworkcontroller.NewGenericController[*imgdom.Image](
			fakeClient,
			ImageFromCR,
			handler,
			&Image{},
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
