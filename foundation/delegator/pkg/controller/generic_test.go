package controller

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"
	"time"

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
//go:generate mockgen -package controller -destination=zz_mock_client_test.go sigs.k8s.io/controller-runtime/pkg/client Client
//go:generate mockgen -package controller -destination=zz_mock_status_writer_test.go sigs.k8s.io/controller-runtime/pkg/client StatusWriter

// TestDomainResource is a dummy implementation of IdentifiableResource for testing.
type TestDomainResource struct {
	Name      string
	Tenant    string
	Workspace string
}

func (r *TestDomainResource) GetName() string      { return r.Name }
func (r *TestDomainResource) GetTenant() string    { return r.Tenant }
func (r *TestDomainResource) GetWorkspace() string { return r.Workspace }

// TestK8sResourceStatus is a dummy status for testing.
type TestK8sResourceStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// TestK8sResource is a dummy implementation of client.Object for testing.
type TestK8sResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              string                `json:"spec"`
	Status            TestK8sResourceStatus `json:"status,omitempty"`
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
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
		for i := range in.Status.Conditions {
			in.Status.Conditions[i].DeepCopyInto(&out.Status.Conditions[i])
		}
	}
}

// BadUnstructuredResource contains a field that cannot be converted to unstructured (chan).
type BadUnstructuredResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	BadField          chan int `json:"badField"`
}

func (in *BadUnstructuredResource) DeepCopyObject() runtime.Object {
	return in // Not used in this test scenario
}

// BadStatusResource contains a status field that is not a map/struct, causing SetNestedSlice to fail.
type BadStatusResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            string `json:"status"`
}

func (in *BadStatusResource) DeepCopyObject() runtime.Object {
	out := *in
	return &out
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

		//
		// And a Kubernetes which the schema contains the above resource
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(k8sRes).
			Build()

		//
		// And a resource handler which will succeed
		mockHandler := NewMockPluginHandler[*TestDomainResource](ctrl)
		mockHandler.EXPECT().HandleReconcile(gomock.Any(), domainRes).Return(nil).Times(1)

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a generic controller using these elements
		reconciler := NewGenericController[*TestDomainResource](
			fakeClient,
			converter,
			mockHandler,
			prototype,
			0,
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
		// And a resource handler
		mockHandler := NewMockPluginHandler[*TestDomainResource](ctrl)

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a generic controller using these elements
		reconciler := NewGenericController[*TestDomainResource](
			fakeClient,
			converter,
			mockHandler,
			prototype,
			0,
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

		//
		// And no error logs should be produced
		require.Empty(t, buf.String())
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

		//
		// And a Kubernetes which the schema contains the above resource
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(k8sRes).
			WithStatusSubresource(k8sRes).
			Build()

		//
		// And a resource handler
		mockHandler := NewMockPluginHandler[*TestDomainResource](ctrl)

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a generic controller using these elements
		reconciler := NewGenericController[*TestDomainResource](
			fakeClient,
			converter,
			mockHandler,
			prototype,
			0,
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

		// And the status should be updated to indicate the conversion error
		var updatedRes TestK8sResource
		err = fakeClient.Get(t.Context(), client.ObjectKey{Name: testName, Namespace: testNamespace}, &updatedRes)
		require.NoError(t, err)
		require.Len(t, updatedRes.Status.Conditions, 1)
		require.Equal(t, "ConversionFailed", updatedRes.Status.Conditions[0].Type)
		require.Equal(t, metav1.ConditionTrue, updatedRes.Status.Conditions[0].Status)
		require.Equal(t, "conversion error", updatedRes.Status.Conditions[0].Message)

		//
		// And the error should be logged
		require.Contains(t, buf.String(), "failed to convert k8s object to domain resource")
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

		//
		// And a Kubernetes which the schema contains the above resource
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(k8sRes).
			Build()

		//
		// And a resource handler which will return an error
		mockHandler := NewMockPluginHandler[*TestDomainResource](ctrl)
		mockHandler.EXPECT().HandleReconcile(gomock.Any(), domainRes).Return(errHandler).Times(1)

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a generic controller using these elements
		reconciler := NewGenericController[*TestDomainResource](
			fakeClient,
			converter,
			mockHandler,
			prototype,
			5*time.Minute,
			logger,
		)

		//
		// When we try to reconcile the resource
		req := k8srt.Request{NamespacedName: client.ObjectKey{Name: testName, Namespace: testNamespace}}
		res, err := reconciler.Reconcile(t.Context(), req)

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

func TestGenericController_updateStatusCondition(t *testing.T) {
	const (
		testName      = "test-resource-status"
		testNamespace = "default"
	)

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(testGVK.GroupVersion(), &TestK8sResource{})

	prototype := &TestK8sResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       testGVK.Kind,
			APIVersion: testGVK.GroupVersion().String(),
		},
	}

	t.Run("should add a new condition when none exist", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with no conditions
		k8sRes := &TestK8sResource{
			TypeMeta: metav1.TypeMeta{
				Kind:       testGVK.Kind,
				APIVersion: testGVK.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
		}

		//
		// And a Kubernetes which the schema contains the above resource and its status subresource
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(k8sRes).
			WithStatusSubresource(k8sRes).
			Build()

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a generic controller using these elements
		reconciler := NewGenericController[*TestDomainResource](
			fakeClient,
			nil, // k8sToDomain not needed
			nil, // handler not needed
			prototype,
			0,
			logger,
		)

		//
		// And a condition to be added
		condition := metav1.Condition{
			Type:               "TestCondition",
			Status:             metav1.ConditionTrue,
			Reason:             "TestReason",
			Message:            "TestMessage",
			LastTransitionTime: metav1.Now(),
		}

		//
		// When we update the status condition
		reconciler.updateStatusCondition(t.Context(), k8sRes, condition)

		//
		// Then the condition should be added
		var updatedRes TestK8sResource
		err := fakeClient.Get(t.Context(), client.ObjectKey{Name: testName, Namespace: testNamespace}, &updatedRes)
		require.NoError(t, err)
		require.Len(t, updatedRes.Status.Conditions, 1)
		require.Equal(t, condition.Type, updatedRes.Status.Conditions[0].Type)
		require.Equal(t, condition.Status, updatedRes.Status.Conditions[0].Status)
		require.Equal(t, condition.Reason, updatedRes.Status.Conditions[0].Reason)
		require.Equal(t, condition.Message, updatedRes.Status.Conditions[0].Message)

		//
		// And no error logs should be produced
		require.Empty(t, buf.String())
	})

	t.Run("should update an existing condition", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with an existing condition
		existingCondition := metav1.Condition{
			Type:               "TestCondition",
			Status:             metav1.ConditionFalse,
			Reason:             "OldReason",
			Message:            "OldMessage",
			LastTransitionTime: metav1.NewTime(metav1.Now().Add(-1 * time.Hour)),
		}

		k8sRes := &TestK8sResource{
			TypeMeta: metav1.TypeMeta{
				Kind:       testGVK.Kind,
				APIVersion: testGVK.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Status: TestK8sResourceStatus{
				Conditions: []metav1.Condition{existingCondition},
			},
		}

		//
		// And a Kubernetes which the schema contains the above resource and its status subresource
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(k8sRes).
			WithStatusSubresource(k8sRes).
			Build()

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a generic controller using these elements
		reconciler := NewGenericController[*TestDomainResource](
			fakeClient,
			nil,
			nil,
			prototype,
			0,
			logger,
		)

		//
		// And a new condition with the same type
		newCondition := metav1.Condition{
			Type:               "TestCondition",
			Status:             metav1.ConditionTrue,
			Reason:             "NewReason",
			Message:            "NewMessage",
			LastTransitionTime: metav1.Now(),
		}

		//
		// When we update the status condition with the same type
		reconciler.updateStatusCondition(t.Context(), k8sRes, newCondition)

		//
		// Then the condition should be updated
		var updatedRes TestK8sResource
		err := fakeClient.Get(t.Context(), client.ObjectKey{Name: testName, Namespace: testNamespace}, &updatedRes)
		require.NoError(t, err)
		require.Len(t, updatedRes.Status.Conditions, 1)
		require.Equal(t, newCondition.Type, updatedRes.Status.Conditions[0].Type)
		require.Equal(t, newCondition.Status, updatedRes.Status.Conditions[0].Status)
		require.Equal(t, newCondition.Reason, updatedRes.Status.Conditions[0].Reason)
		require.Equal(t, newCondition.Message, updatedRes.Status.Conditions[0].Message)

		//
		// And no error logs should be produced
		require.Empty(t, buf.String())
	})

	t.Run("should preserve other conditions", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource with an existing different condition
		otherCondition := metav1.Condition{
			Type:               "OtherCondition",
			Status:             metav1.ConditionTrue,
			Reason:             "OtherReason",
			Message:            "OtherMessage",
			LastTransitionTime: metav1.Now(),
		}

		k8sRes := &TestK8sResource{
			TypeMeta: metav1.TypeMeta{
				Kind:       testGVK.Kind,
				APIVersion: testGVK.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Status: TestK8sResourceStatus{
				Conditions: []metav1.Condition{otherCondition},
			},
		}

		//
		// And a Kubernetes which the schema contains the above resource and its status subresource
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(k8sRes).
			WithStatusSubresource(k8sRes).
			Build()

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a generic controller using these elements
		reconciler := NewGenericController[*TestDomainResource](
			fakeClient,
			nil,
			nil,
			prototype,
			0,
			logger,
		)

		//
		// And a new condition
		newCondition := metav1.Condition{
			Type:               "TestCondition",
			Status:             metav1.ConditionTrue,
			Reason:             "TestReason",
			Message:            "TestMessage",
			LastTransitionTime: metav1.Now(),
		}

		//
		// When we update the status condition
		reconciler.updateStatusCondition(t.Context(), k8sRes, newCondition)

		//
		// Then both conditions should be present
		var updatedRes TestK8sResource
		err := fakeClient.Get(t.Context(), client.ObjectKey{Name: testName, Namespace: testNamespace}, &updatedRes)
		require.NoError(t, err)
		require.Len(t, updatedRes.Status.Conditions, 2)

		//
		// And the conditions should have the expected types
		require.Equal(t, otherCondition.Type, updatedRes.Status.Conditions[0].Type)
		require.Equal(t, newCondition.Type, updatedRes.Status.Conditions[1].Type)

		//
		// And no error logs should be produced
		require.Empty(t, buf.String())
	})

	t.Run("should log error when object conversion to unstructured fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		//
		// Given a resource that cannot be converted to unstructured (contains chan)
		badRes := &BadUnstructuredResource{
			TypeMeta: metav1.TypeMeta{
				Kind:       testGVK.Kind,
				APIVersion: testGVK.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			BadField: make(chan int),
		}

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a generic controller
		reconciler := NewGenericController[*TestDomainResource](
			nil, // client not needed for this failure
			nil,
			nil,
			prototype,
			0,
			logger,
		)

		condition := metav1.Condition{
			Type:   "TestCondition",
			Status: metav1.ConditionTrue,
		}

		//
		// When we update the status condition
		reconciler.updateStatusCondition(t.Context(), badRes, condition)

		//
		// Then it should log the error
		require.Contains(t, buf.String(), "failed to convert object to unstructured for status update")
	})

	t.Run("should log error when setting conditions in unstructured object fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a resource where status is not a map (string)
		badRes := &BadStatusResource{
			TypeMeta: metav1.TypeMeta{
				Kind:       testGVK.Kind,
				APIVersion: testGVK.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Status: "active",
		}

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a generic controller
		reconciler := NewGenericController[*TestDomainResource](
			nil, // client not needed for this failure
			nil,
			nil,
			prototype,
			0,
			logger,
		)

		condition := metav1.Condition{
			Type:   "TestCondition",
			Status: metav1.ConditionTrue,
		}

		//
		// When we update the status condition
		reconciler.updateStatusCondition(t.Context(), badRes, condition)

		//
		// Then it should log the error
		require.Contains(t, buf.String(), "failed to set conditions in unstructured object")
	})

	t.Run("should log error when status update fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a valid resource
		k8sRes := &TestK8sResource{
			TypeMeta: metav1.TypeMeta{
				Kind:       testGVK.Kind,
				APIVersion: testGVK.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
		}

		//
		// And a client that fails on status update
		errUpdate := errors.New("update failed")
		mockClient := NewMockClient(ctrl)
		mockStatusWriter := NewMockStatusWriter(ctrl)
		mockClient.EXPECT().Status().Return(mockStatusWriter).Times(1)
		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errUpdate).Times(1)

		//
		// And a logger to capture output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))

		//
		// And a generic controller
		reconciler := NewGenericController[*TestDomainResource](
			mockClient,
			nil,
			nil,
			prototype,
			0,
			logger,
		)

		condition := metav1.Condition{
			Type:   "TestCondition",
			Status: metav1.ConditionTrue,
		}

		//
		// When we update the status condition
		reconciler.updateStatusCondition(t.Context(), k8sRes, condition)

		//
		// Then it should log the error
		require.Contains(t, buf.String(), "failed to update status")
		require.Contains(t, buf.String(), "update failed")
	})
}
