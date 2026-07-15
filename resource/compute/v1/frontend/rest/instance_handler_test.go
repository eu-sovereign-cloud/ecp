package rest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// fakeInstanceRepo is a hand-rolled ReaderRepo+WriterRepo for exercising the compute Handler over
// HTTP. Load returns a copy of loadResult (or loadErr); Create/Update capture the written domain.
type fakeInstanceRepo struct {
	loadResult *instancedom.Instance
	loadErr    error
	writeErr   error
	written    *instancedom.Instance // captured by Create/Update
}

var (
	_ persistencepkg.ReaderRepo[*instancedom.Instance] = (*fakeInstanceRepo)(nil)
	_ persistencepkg.WriterRepo[*instancedom.Instance] = (*fakeInstanceRepo)(nil)
)

func (f *fakeInstanceRepo) List(context.Context, resource.ListFilter, *[]*instancedom.Instance) (*string, error) {
	return nil, nil
}

func (f *fakeInstanceRepo) Load(_ context.Context, m **instancedom.Instance) error {
	if f.loadErr != nil {
		return f.loadErr
	}
	cp := *f.loadResult
	*m = &cp
	return nil
}

func (f *fakeInstanceRepo) Delete(context.Context, *instancedom.Instance) error { return nil }

func (f *fakeInstanceRepo) Create(_ context.Context, m *instancedom.Instance) (**instancedom.Instance, error) {
	f.written = m
	if f.writeErr != nil {
		return nil, f.writeErr
	}
	return &m, nil
}

func (f *fakeInstanceRepo) Update(_ context.Context, m *instancedom.Instance) (**instancedom.Instance, error) {
	f.written = m
	if f.writeErr != nil {
		return nil, f.writeErr
	}
	return &m, nil
}

func (f *fakeInstanceRepo) UpdateStatus(_ context.Context, m *instancedom.Instance) (**instancedom.Instance, error) {
	return &m, nil
}

func newTestHandler(repo *fakeInstanceRepo) *Handler {
	return &Handler{
		InstanceReader: repo,
		InstanceWriter: repo,
		Logger:         slog.Default(),
	}
}

func activeInstance(powerState instancedom.PowerState) *instancedom.Instance {
	inst := &instancedom.Instance{
		Status: &instancedom.InstanceStatus{
			Status:     commondomain.Status{State: commondomain.ResourceStateActive},
			PowerState: powerState,
		},
	}
	inst.Name = "inst1"
	inst.Tenant = "t1"
	inst.Workspace = "w1"
	return inst
}

const (
	testTenant    = "t1"
	testWorkspace = "w1"
	testName      = "inst1"
)

func TestHandler_PowerPreconditions(t *testing.T) {
	tests := []struct {
		name       string
		power      instancedom.PowerState
		invoke     func(h *Handler, w http.ResponseWriter, r *http.Request)
		wantStatus int
		wantWrite  bool
	}{
		{
			name:  "start rejected when already on",
			power: instancedom.PowerStateOn,
			invoke: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.StartInstance(w, r, testTenant, testWorkspace, testName, sdkcompute.StartInstanceParams{})
			},
			wantStatus: http.StatusConflict,
		},
		{
			name:  "stop rejected when already off",
			power: instancedom.PowerStateOff,
			invoke: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.StopInstance(w, r, testTenant, testWorkspace, testName, sdkcompute.StopInstanceParams{})
			},
			wantStatus: http.StatusConflict,
		},
		{
			name:  "restart rejected when powered off",
			power: instancedom.PowerStateOff,
			invoke: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.RestartInstance(w, r, testTenant, testWorkspace, testName, sdkcompute.RestartInstanceParams{})
			},
			wantStatus: http.StatusConflict,
		},
		{
			name:  "start accepted when off",
			power: instancedom.PowerStateOff,
			invoke: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.StartInstance(w, r, testTenant, testWorkspace, testName, sdkcompute.StartInstanceParams{})
			},
			wantStatus: http.StatusAccepted,
			wantWrite:  true,
		},
		{
			name:  "stop accepted when on",
			power: instancedom.PowerStateOn,
			invoke: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.StopInstance(w, r, testTenant, testWorkspace, testName, sdkcompute.StopInstanceParams{})
			},
			wantStatus: http.StatusAccepted,
			wantWrite:  true,
		},
		{
			name:  "restart accepted when on",
			power: instancedom.PowerStateOn,
			invoke: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.RestartInstance(w, r, testTenant, testWorkspace, testName, sdkcompute.RestartInstanceParams{})
			},
			wantStatus: http.StatusAccepted,
			wantWrite:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeInstanceRepo{loadResult: activeInstance(tc.power)}
			h := newTestHandler(repo)

			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			tc.invoke(h, rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantWrite {
				require.NotNil(t, repo.written, "an accepted power op must persist intent")
			} else {
				require.Nil(t, repo.written, "a rejected power op must not write")
			}
		})
	}
}

func TestHandler_PowerIntentPersisted(t *testing.T) {
	t.Run("start sets desired power state on", func(t *testing.T) {
		repo := &fakeInstanceRepo{loadResult: activeInstance(instancedom.PowerStateOff)}
		h := newTestHandler(repo)
		rec := httptest.NewRecorder()
		h.StartInstance(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil), testTenant, testWorkspace, testName, sdkcompute.StartInstanceParams{})

		require.Equal(t, http.StatusAccepted, rec.Code)
		require.Equal(t, instancedom.PowerStateOn, repo.written.DesiredPowerState)
	})

	t.Run("restart sets a fresh id and the initial phase", func(t *testing.T) {
		repo := &fakeInstanceRepo{loadResult: activeInstance(instancedom.PowerStateOn)}
		h := newTestHandler(repo)
		rec := httptest.NewRecorder()
		h.RestartInstance(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil), testTenant, testWorkspace, testName, sdkcompute.RestartInstanceParams{})

		require.Equal(t, http.StatusAccepted, rec.Code)
		require.NotEmpty(t, repo.written.RestartID)
		require.Equal(t, instancedom.RestartPhasePowerOff, repo.written.RestartPhase)
	})

	t.Run("If-Unmodified-Since is applied as the resource version precondition", func(t *testing.T) {
		repo := &fakeInstanceRepo{loadResult: activeInstance(instancedom.PowerStateOff)}
		h := newTestHandler(repo)
		version := sdkschema.IfUnmodifiedSince(7)
		rec := httptest.NewRecorder()
		h.StartInstance(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil), testTenant, testWorkspace, testName,
			sdkcompute.StartInstanceParams{IfUnmodifiedSince: &version})

		require.Equal(t, http.StatusAccepted, rec.Code)
		require.Equal(t, "7", repo.written.ResourceVersion, "the update must carry the precondition version")
	})

	t.Run("without If-Unmodified-Since the update is fire-and-forget (no version)", func(t *testing.T) {
		loaded := activeInstance(instancedom.PowerStateOff)
		loaded.ResourceVersion = "99" // the loaded object has a current version...
		repo := &fakeInstanceRepo{loadResult: loaded}
		h := newTestHandler(repo)
		rec := httptest.NewRecorder()
		h.StartInstance(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil), testTenant, testWorkspace, testName, sdkcompute.StartInstanceParams{})

		require.Equal(t, http.StatusAccepted, rec.Code)
		require.Empty(t, repo.written.ResourceVersion, "...but it is cleared so the write is fire-and-forget")
	})
}

func TestHandler_PowerOpErrors(t *testing.T) {
	t.Run("not found returns 404", func(t *testing.T) {
		repo := &fakeInstanceRepo{loadErr: kernel.ErrNotFound}
		h := newTestHandler(repo)
		rec := httptest.NewRecorder()
		h.StartInstance(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil), testTenant, testWorkspace, testName, sdkcompute.StartInstanceParams{})

		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Nil(t, repo.written)
	})

	t.Run("If-Unmodified-Since mismatch surfaces 412 Precondition Failed", func(t *testing.T) {
		// The framework maps a stale-resourceVersion conflict to KindPreconditionFailed; the
		// handler must surface it as 412 (not 409), like the other write ops.
		repo := &fakeInstanceRepo{
			loadResult: activeInstance(instancedom.PowerStateOff),
			writeErr:   kernel.NewError(kernel.KindPreconditionFailed, errors.New("resource version mismatch")),
		}
		h := newTestHandler(repo)
		version := sdkschema.IfUnmodifiedSince(3)
		rec := httptest.NewRecorder()
		h.StartInstance(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil), testTenant, testWorkspace, testName,
			sdkcompute.StartInstanceParams{IfUnmodifiedSince: &version})

		require.Equal(t, http.StatusPreconditionFailed, rec.Code)
	})

	t.Run("non-active instance is rejected with 409", func(t *testing.T) {
		inst := activeInstance(instancedom.PowerStateOff)
		inst.Status.State = commondomain.ResourceStateCreating
		repo := &fakeInstanceRepo{loadResult: inst}
		h := newTestHandler(repo)
		rec := httptest.NewRecorder()
		h.StartInstance(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil), testTenant, testWorkspace, testName, sdkcompute.StartInstanceParams{})

		require.Equal(t, http.StatusConflict, rec.Code)
		require.Nil(t, repo.written)
	})
}

func TestHandler_CreateOrUpdatePreservesPowerIntent(t *testing.T) {
	// An ordinary PUT carries a body with no power intent; the handler must carry forward the
	// existing internal power state rather than erasing it.
	existing := activeInstance(instancedom.PowerStateOn)
	existing.DesiredPowerState = instancedom.PowerStateOn
	existing.RestartID = "rid-1"
	existing.RestartPhase = instancedom.RestartPhasePowerOff
	repo := &fakeInstanceRepo{loadResult: existing}
	h := newTestHandler(repo)

	body, err := json.Marshal(sdkschema.Instance{Spec: sdkschema.InstanceSpec{Zone: "zone-a"}})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/", strings.NewReader(string(body)))
	h.CreateOrUpdateInstance(rec, req, testTenant, testWorkspace, testName, sdkcompute.CreateOrUpdateInstanceParams{})

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, repo.written)
	require.Equal(t, instancedom.PowerStateOn, repo.written.DesiredPowerState, "desired power state must be preserved")
	require.Equal(t, "rid-1", repo.written.RestartID, "restart id must be preserved")
	require.Equal(t, instancedom.RestartPhasePowerOff, repo.written.RestartPhase, "restart phase must be preserved")
}
