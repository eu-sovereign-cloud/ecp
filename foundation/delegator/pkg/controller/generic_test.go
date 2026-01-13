package controller

import (
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8srt "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

//go:generate mockgen -package controller -destination=zz_mock_plugin_handler_test.go github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port PluginHandler

// TestDomainResource is a dummy implementation of IdentifiableResource for testing.
type TestDomainResource struct {
	Name      string
	Tenant    string
	Workspace string
}

func (r *TestDomainResource) GetName() string      { return r.Name }
func (r *TestDomainResource) GetTenant() string    { return r.Tenant }
func (r *TestDomainResource) GetWorkspace() string { return r.Workspace }

// TestK8sResource is a dummy implementation of client.Object for testing.
type TestK8sResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              string `json:"spec"`
}

func (in *TestK8sResource) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(TestK8sResource)
	in.DeepCopyInto(out)
	return out
}

func (in *TestK8sResource) DeepCopyInto(out *TestK8sResource) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
}

var (
	testGVK = schema.GroupVersionKind{
		Group:   "test.ecp.io",
		Version: "v1",
		Kind:    "TestK8sResource",
	}
)

func TestGenericController_Reconcile(t *testing.T) {
	const (
		testName      = "test-resource"
		testNamespace = "default"
		validSpec     = "valid"
		invalidSpec   = "invalid"
	)

	var (
		errConversion = errors.New("conversion error")
		errHandler    = errors.New("handler failed")
	)

	// Common Setup
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(testGVK.GroupVersion(), &TestK8sResource{})

	domainRes := &TestDomainResource{Name: testName}

	// Converter function (acting as the Adapter)
	converter := func(obj client.Object) (*TestDomainResource, error) {
		kObj, ok := obj.(*TestK8sResource)
		if !ok {
			return nil, errors.New("invalid type")
		}
		if kObj.Spec == invalidSpec {
			return nil, errConversion
		}
		return domainRes, nil
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	prototype := &TestK8sResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       testGVK.Kind,
			APIVersion: testGVK.GroupVersion().String(),
		},
	}

	t.Run("should successfully reconcile a valid resource", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a valid K8s resource
		k8sRes := &TestK8sResource{
			TypeMeta: metav1.TypeMeta{
				Kind:       testGVK.Kind,
				APIVersion: testGVK.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: validSpec,
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(k8sRes).
			Build()

		//
		// And a resource handler which will succeed
		mockHandler := NewMockPluginHandler[*TestDomainResource](ctrl)
		mockHandler.EXPECT().HandleReconcile(gomock.Any(), domainRes).Return(nil).Times(1)

		//
		// And a generic controller using these elements
		reconciler := NewGenericController[*TestDomainResource, *TestK8sResource](
			fakeClient,
			converter,
			mockHandler,
			prototype,
			logger,
		)

		//
		// When we try to reconcile the resource
		req := k8srt.Request{NamespacedName: client.ObjectKey{Name: testName, Namespace: testNamespace}}
		res, err := reconciler.Reconcile(t.Context(), req)

		//
		// Then it should succeed
		require.NoError(t, err)
		require.Equal(t, k8srt.Result{}, res)
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
		// And a resource handler
		mockHandler := NewMockPluginHandler[*TestDomainResource](ctrl)

		//
		// And a generic controller using these elements
		reconciler := NewGenericController[*TestDomainResource, *TestK8sResource](
			fakeClient,
			converter,
			mockHandler,
			prototype,
			logger,
		)

		//
		// When we try to reconcile a missing resource
		req := k8srt.Request{NamespacedName: client.ObjectKey{Name: "missing", Namespace: testNamespace}}
		res, err := reconciler.Reconcile(t.Context(), req)

		//
		// Then it should return no error and no result
		require.NoError(t, err)
		require.Equal(t, k8srt.Result{}, res)
	})

	t.Run("should log and ignore when resource conversion fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a K8s resource that will fail conversion
		k8sRes := &TestK8sResource{
			TypeMeta: metav1.TypeMeta{
				Kind:       testGVK.Kind,
				APIVersion: testGVK.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: invalidSpec,
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(k8sRes).
			Build()

		//
		// And a resource handler
		mockHandler := NewMockPluginHandler[*TestDomainResource](ctrl)

		//
		// And a generic controller using these elements
		reconciler := NewGenericController[*TestDomainResource, *TestK8sResource](
			fakeClient,
			converter,
			mockHandler,
			prototype,
			logger,
		)

		//
		// When we try to reconcile the resource
		req := k8srt.Request{NamespacedName: client.ObjectKey{Name: testName, Namespace: testNamespace}}
		res, err := reconciler.Reconcile(t.Context(), req)

		//
		// Then it should return no error (as it is logged and ignored)
		require.NoError(t, err)
		require.Equal(t, k8srt.Result{}, res)
	})

	t.Run("should report an error when handler fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a valid K8s resource
		k8sRes := &TestK8sResource{
			TypeMeta: metav1.TypeMeta{
				Kind:       testGVK.Kind,
				APIVersion: testGVK.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: validSpec,
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(k8sRes).
			Build()

		//
		// And a resource handler which will return an error
		mockHandler := NewMockPluginHandler[*TestDomainResource](ctrl)
		mockHandler.EXPECT().HandleReconcile(gomock.Any(), domainRes).Return(errHandler).Times(1)

		//
		// And a generic controller using these elements
		reconciler := NewGenericController[*TestDomainResource, *TestK8sResource](
			fakeClient,
			converter,
			mockHandler,
			prototype,
			logger,
		)

		//
		// When we try to reconcile the resource
		req := k8srt.Request{NamespacedName: client.ObjectKey{Name: testName, Namespace: testNamespace}}
		_, err := reconciler.Reconcile(t.Context(), req)

		//
		// Then it should return the handler error
		require.ErrorIs(t, err, errHandler)
	})
}
