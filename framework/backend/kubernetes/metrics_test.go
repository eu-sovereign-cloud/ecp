package kubernetes

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

type recordingObserver struct {
	mu  sync.Mutex
	obs []upstreamObservation
}

type upstreamObservation struct {
	resource, group, operation, result string
	d                                  time.Duration
}

func (r *recordingObserver) Observe(resource, group, operation, result string, d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.obs = append(r.obs, upstreamObservation{
		resource: resource, group: group, operation: operation, result: result, d: d,
	})
}

func (r *recordingObserver) snapshot() []upstreamObservation {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]upstreamObservation, len(r.obs))
	copy(out, r.obs)
	return out
}

func TestClassifyUpstreamResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: ResultOK},
		{name: "not found api", err: kerrs.NewNotFound(schema.GroupResource{Resource: "pods"}, "x"), want: ResultNotFound},
		{name: "already exists", err: kerrs.NewAlreadyExists(schema.GroupResource{Resource: "pods"}, "x"), want: ResultAlreadyExists},
		{name: "conflict", err: kerrs.NewConflict(schema.GroupResource{Resource: "pods"}, "x", errors.New("c")), want: ResultConflict},
		{name: "forbidden", err: kerrs.NewForbidden(schema.GroupResource{Resource: "pods"}, "x", errors.New("f")), want: ResultForbidden},
		{name: "deadline", err: context.DeadlineExceeded, want: ResultTimeout},
		{name: "kernel not found", err: kernel.NewError(kernel.KindNotFound, errors.New("missing")), want: ResultNotFound},
		{name: "other", err: errors.New("boom"), want: ResultError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := classifyUpstreamResult(tt.err); got != tt.want {
				t.Errorf("classifyUpstreamResult() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReaderAdapter_Load_ObservesGet(t *testing.T) {
	rec := &recordingObserver{}
	SetUpstreamObserver(rec)
	t.Cleanup(func() { SetUpstreamObserver(nil) })

	// Cluster-scoped path: empty tenant/workspace → namespace "".
	obj := newTestObject("", "rt-1")
	dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), testListKinds(), obj)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	reader := NewReaderAdapter[*testIdentifiable](dynFake, testGVR, logger, func(o client.Object) (*testIdentifiable, error) {
		return &testIdentifiable{name: o.GetName()}, nil
	})

	loaded := &testIdentifiable{name: "rt-1"}
	err := reader.Load(context.Background(), &loaded)
	require.NoError(t, err)

	obs := rec.snapshot()
	require.Len(t, obs, 1)
	require.Equal(t, "routetables", obs[0].resource)
	require.Equal(t, "network.test", obs[0].group)
	require.Equal(t, OpGet, obs[0].operation)
	require.Equal(t, ResultOK, obs[0].result)
	require.GreaterOrEqual(t, obs[0].d, time.Duration(0))
}

func TestReaderAdapter_Load_ObservesNotFound(t *testing.T) {
	rec := &recordingObserver{}
	SetUpstreamObserver(rec)
	t.Cleanup(func() { SetUpstreamObserver(nil) })

	dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), testListKinds())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	reader := NewReaderAdapter[*testIdentifiable](dynFake, testGVR, logger, func(o client.Object) (*testIdentifiable, error) {
		return &testIdentifiable{name: o.GetName()}, nil
	})

	loaded := &testIdentifiable{name: "missing"}
	err := reader.Load(context.Background(), &loaded)
	require.Error(t, err)

	obs := rec.snapshot()
	require.Len(t, obs, 1)
	require.Equal(t, OpGet, obs[0].operation)
	require.Equal(t, ResultNotFound, obs[0].result)
}

func TestReaderAdapter_List_ObservesList(t *testing.T) {
	rec := &recordingObserver{}
	SetUpstreamObserver(rec)
	t.Cleanup(func() { SetUpstreamObserver(nil) })

	workspaceNS := ComputeNamespace(&kernelresource.Scope{Tenant: "t1", Workspace: "w1"})
	obj := newTestObject(workspaceNS, "in-workspace-ns")
	dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), testListKinds(), obj)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	reader := NewReaderAdapter[*testIdentifiable](dynFake, testGVR, logger, func(o client.Object) (*testIdentifiable, error) {
		return &testIdentifiable{name: o.GetName()}, nil
	})

	var out []*testIdentifiable
	_, err := reader.List(context.Background(), kernelresource.ListParams{
		Scope: kernelresource.Scope{Tenant: "t1", Workspace: "w1"},
	}, &out)
	require.NoError(t, err)

	obs := rec.snapshot()
	require.Len(t, obs, 1)
	require.Equal(t, OpList, obs[0].operation)
	require.Equal(t, ResultOK, obs[0].result)
	require.Equal(t, testGVR.Resource, obs[0].resource)
}

func TestWriterAdapter_Create_ObservesCreate(t *testing.T) {
	rec := &recordingObserver{}
	SetUpstreamObserver(rec)
	t.Cleanup(func() { SetUpstreamObserver(nil) })

	dynFake := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), testListKinds())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	writer := NewWriterAdapter[*testIdentifiable](
		dynFake,
		testGVR,
		logger,
		func(d *testIdentifiable) (client.Object, error) {
			return newTestObject("", d.name), nil
		},
		func(o client.Object) (*testIdentifiable, error) {
			return &testIdentifiable{name: o.GetName()}, nil
		},
	)

	created, err := writer.Create(context.Background(), &testIdentifiable{name: "new-rt"})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, "new-rt", (*created).name)

	obs := rec.snapshot()
	require.Len(t, obs, 1)
	require.Equal(t, OpCreate, obs[0].operation)
	require.Equal(t, ResultOK, obs[0].result)
}

func TestCreateNamespace_ObservesNamespaces(t *testing.T) {
	rec := &recordingObserver{}
	SetUpstreamObserver(rec)
	t.Cleanup(func() { SetUpstreamObserver(nil) })

	cs := k8sfake.NewClientset()
	created, err := CreateNamespace(context.Background(), cs, "ns-observe", map[string]string{"k": "v"})
	require.NoError(t, err)
	require.True(t, created)

	obs := rec.snapshot()
	require.Len(t, obs, 1)
	require.Equal(t, "namespaces", obs[0].resource)
	require.Equal(t, "core", obs[0].group)
	require.Equal(t, OpCreate, obs[0].operation)
	require.Equal(t, ResultOK, obs[0].result)
}
