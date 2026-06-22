package crossplane

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpconditions "github.com/crossplane/crossplane-runtime/v2/pkg/conditions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	delegator "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
)

// compile-time check: testCR must satisfy the interface used by base methods.
var _ xpconditions.ObjectWithConditions = (*testCR)(nil)

// testCR is a minimal Crossplane CR stub for testing base methods.
type testCR struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	v1.ConditionedStatus
}

func (t *testCR) DeepCopyObject() runtime.Object {
	cp := *t
	return &cp
}

// fakeClient stubs only the client methods exercised by base.
// Unimplemented methods panic via the embedded nil interface.
type fakeClient struct {
	client.Client
	createErr error
	deleteErr error
	updateErr error
	getFunc   func(client.Object) error
}

func (f *fakeClient) Create(_ context.Context, _ client.Object, _ ...client.CreateOption) error {
	return f.createErr
}

func (f *fakeClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	if f.getFunc != nil {
		return f.getFunc(obj)
	}
	return nil
}

func (f *fakeClient) Delete(_ context.Context, _ client.Object, _ ...client.DeleteOption) error {
	return f.deleteErr
}

func (f *fakeClient) Update(_ context.Context, _ client.Object, _ ...client.UpdateOption) error {
	return f.updateErr
}

func discardBase(fc *fakeClient) *base {
	return &base{client: fc, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func newTestObj() *testCR {
	return &testCR{
		TypeMeta:   metav1.TypeMeta{Kind: "TestResource"},
		ObjectMeta: metav1.ObjectMeta{Name: "obj", Namespace: "ns", Generation: 1},
	}
}

func alreadyExistsErr() error { return apierrors.NewAlreadyExists(schema.GroupResource{}, "obj") }
func notFoundErr() error      { return apierrors.NewNotFound(schema.GroupResource{}, "obj") }

func TestCreateCR(t *testing.T) {
	errAPI := errors.New("api error")

	tests := []struct {
		name         string
		fc           *fakeClient
		wantErr      error  // checked with errors.Is; nil means expect no error
		wantContains string // non-empty: expect error whose message contains this
	}{
		{
			name:    "create API error propagates",
			fc:      &fakeClient{createErr: errAPI},
			wantErr: errAPI,
		},
		{
			name:    "created successfully waits for ready",
			fc:      &fakeClient{},
			wantErr: delegator.ErrStillProcessing,
		},
		{
			name: "already exists not yet ready waits",
			fc:   &fakeClient{createErr: alreadyExistsErr()},
			// getFunc nil → Get returns nil with no conditions → Ready=Unknown
			wantErr: delegator.ErrStillProcessing,
		},
		{
			name: "already exists and ready returns nil",
			fc: &fakeClient{
				createErr: alreadyExistsErr(),
				getFunc: func(obj client.Object) error {
					obj.(*testCR).SetConditions(v1.Available())
					return nil
				},
			},
		},
		{
			name: "already exists get error propagates",
			fc: &fakeClient{
				createErr: alreadyExistsErr(),
				getFunc:   func(_ client.Object) error { return errAPI },
			},
			wantErr: errAPI,
		},
		{
			name: "already exists ReconcileError surfaces provider failure",
			fc: &fakeClient{
				createErr: alreadyExistsErr(),
				getFunc: func(obj client.Object) error {
					obj.(*testCR).SetConditions(v1.ReconcileError(errors.New("disk quota exceeded")))
					return nil
				},
			},
			wantContains: "provider failed to reconcile",
		},
		{
			name: "already exists with deletion timestamp waits",
			fc: &fakeClient{
				createErr: alreadyExistsErr(),
				getFunc: func(obj client.Object) error {
					now := metav1.NewTime(time.Now())
					obj.(*testCR).DeletionTimestamp = &now
					return nil
				},
			},
			wantErr: delegator.ErrStillProcessing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := discardBase(tt.fc).createCR(context.Background(), newTestObj())
			assertErr(t, err, tt.wantErr, tt.wantContains)
		})
	}
}

func TestDeleteCR(t *testing.T) {
	errAPI := errors.New("api error")

	tests := []struct {
		name         string
		fc           *fakeClient
		wantErr      error
		wantContains string
	}{
		{
			name:    "resource already gone returns nil",
			fc:      &fakeClient{deleteErr: notFoundErr()},
			wantErr: nil,
		},
		{
			name:    "delete API error propagates",
			fc:      &fakeClient{deleteErr: errAPI},
			wantErr: errAPI,
		},
		{
			name: "resource gone between delete and get returns nil",
			fc: &fakeClient{
				getFunc: func(_ client.Object) error { return notFoundErr() },
			},
			wantErr: nil,
		},
		{
			name: "get error after delete propagates",
			fc: &fakeClient{
				getFunc: func(_ client.Object) error { return errAPI },
			},
			wantErr: errAPI,
		},
		{
			name: "ReconcileError during deletion surfaces provider failure",
			fc: &fakeClient{
				getFunc: func(obj client.Object) error {
					obj.(*testCR).SetConditions(v1.ReconcileError(errors.New("volume in use")))
					return nil
				},
			},
			wantContains: "provider failed to reconcile",
		},
		{
			name:    "resource still exists waits for deletion",
			fc:      &fakeClient{},
			wantErr: delegator.ErrStillProcessing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := discardBase(tt.fc).deleteCR(context.Background(), newTestObj())
			assertErr(t, err, tt.wantErr, tt.wantContains)
		})
	}
}

func TestCheckExisting(t *testing.T) {
	errAPI := errors.New("api error")

	tests := []struct {
		name         string
		fc           *fakeClient
		wantErr      error
		wantContains string
	}{
		{
			name:    "get error propagates",
			fc:      &fakeClient{getFunc: func(_ client.Object) error { return errAPI }},
			wantErr: errAPI,
		},
		{
			name: "ReconcileError surfaces provider failure",
			fc: &fakeClient{
				getFunc: func(obj client.Object) error {
					obj.(*testCR).SetConditions(v1.ReconcileError(errors.New("quota exceeded")))
					return nil
				},
			},
			wantContains: "provider failed to reconcile",
		},
		{
			name: "deletion timestamp returns ErrStillProcessing",
			fc: &fakeClient{
				getFunc: func(obj client.Object) error {
					now := metav1.NewTime(time.Now())
					obj.(*testCR).DeletionTimestamp = &now
					return nil
				},
			},
			wantErr: delegator.ErrStillProcessing,
		},
		{
			name:    "not yet ready returns ErrStillProcessing",
			fc:      &fakeClient{},
			wantErr: delegator.ErrStillProcessing,
		},
		{
			name: "ready with no ObservedGeneration returns nil",
			fc: &fakeClient{
				getFunc: func(obj client.Object) error {
					obj.(*testCR).SetConditions(v1.Available()) // ObservedGeneration stays 0
					return nil
				},
			},
		},
		{
			name: "ready with matching ObservedGeneration returns nil",
			fc: &fakeClient{
				getFunc: func(obj client.Object) error {
					cr := obj.(*testCR)
					cr.SetConditions(v1.Available().WithObservedGeneration(cr.Generation))
					return nil
				},
			},
		},
		{
			name: "ready with stale ObservedGeneration returns ErrStillProcessing",
			fc: &fakeClient{
				getFunc: func(obj client.Object) error {
					cr := obj.(*testCR)
					// Simulate spec update: generation bumped to 2, condition still reflects gen 1.
					cr.Generation = 2
					cr.SetConditions(v1.Available().WithObservedGeneration(1))
					return nil
				},
			},
			wantErr: delegator.ErrStillProcessing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := discardBase(tt.fc).checkExisting(context.Background(), newTestObj())
			assertErr(t, err, tt.wantErr, tt.wantContains)
		})
	}
}

// assertErr checks err against wantIs (errors.Is) or wantContains (substring),
// falling through to nil-check when both are empty.
func assertErr(t *testing.T, got, wantIs error, wantContains string) {
	t.Helper()
	switch {
	case wantIs != nil:
		if !errors.Is(got, wantIs) {
			t.Fatalf("want error %v, got %v", wantIs, got)
		}
	case wantContains != "":
		if got == nil {
			t.Fatal("want error, got nil")
		}
		if !strings.Contains(got.Error(), wantContains) {
			t.Fatalf("want error containing %q, got %v", wantContains, got)
		}
	default:
		if got != nil {
			t.Fatalf("want nil error, got %v", got)
		}
	}
}
