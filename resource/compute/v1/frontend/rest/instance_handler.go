package rest

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// ListInstances handles GET /v1/tenants/{tenant}/workspaces/{workspace}/instances.
func (h *Handler) ListInstances(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdkcompute.ListInstancesParams) {
	logger := h.Logger.With("provider", "compute", "resource", "instance")
	frest.HandleList(w, r, logger, instanceListParamsFromAPI(params, tenant, workspace), frest.ListerFromRepo(h.InstanceReader), instanceIteratorToAPI)
}

// DeleteInstance handles DELETE /v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}.
func (h *Handler) DeleteInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.DeleteInstanceParams) {
	logger := h.Logger.With("provider", "compute", "resource", "instance", "name", name)
	id := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	if params.IfUnmodifiedSince != nil {
		id.Version = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.InstanceWriter, newInstanceWithIdentity))
}

// GetInstance handles GET /v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}.
func (h *Handler) GetInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "compute", "resource", "instance", "name", name)
	ir := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.InstanceReader, newInstanceWithIdentity), instanceToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateInstance handles PUT /v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}.
func (h *Handler) CreateOrUpdateInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.CreateOrUpdateInstanceParams) {
	logger := h.Logger.With("provider", "compute", "resource", "instance", "name", name)
	id := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	if params.IfUnmodifiedSince != nil {
		id.Version = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.Instance, *instancedom.Instance, *sdkschema.Instance]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.InstanceWriter),
		Updater: frest.UpdaterFromRepo(h.InstanceWriter),
		APIToDomain: func(sdk sdkschema.Instance, p persistencepkg.IdentifiableResource) *instancedom.Instance {
			dom := instanceFromAPI(sdk, p.(*resource.Identity), region)
			// Power intent (desired power state, in-flight restart) is controller-managed
			// internal state, not part of the API body. Carry it forward from the existing
			// object so an ordinary spec/label update does not erase a pending power op or
			// restart phase. (On create, Load returns not-found and the fields stay empty.)
			existing := newInstanceWithIdentity(p)
			if err := h.InstanceReader.Load(r.Context(), &existing); err == nil {
				dom.DesiredPowerState = existing.DesiredPowerState
				dom.RestartID = existing.RestartID
				dom.RestartPhase = existing.RestartPhase
			}
			return dom
		},
		DomainToAPI: instanceToAPIWithVerb(http.MethodPut),
	})
}

// StartInstance handles POST /v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}/start.
func (h *Handler) StartInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.StartInstanceParams) {
	logger := h.Logger.With("provider", "compute", "resource", "instance", "name", name, "op", "start")
	inst, ok := h.loadActiveInstance(w, r, logger, tenant, workspace, name)
	if !ok {
		return
	}
	// start is only valid while powered off (409 otherwise, per spec).
	if !h.requirePowerState(w, r, logger, inst, instancedom.PowerStateOff) {
		return
	}
	inst.DesiredPowerState = instancedom.PowerStateOn
	h.applyPowerIntent(w, r, logger, inst, ifUnmodifiedSinceVersion(params.IfUnmodifiedSince))
}

// StopInstance handles POST /v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}/stop.
func (h *Handler) StopInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.StopInstanceParams) {
	logger := h.Logger.With("provider", "compute", "resource", "instance", "name", name, "op", "stop")
	inst, ok := h.loadActiveInstance(w, r, logger, tenant, workspace, name)
	if !ok {
		return
	}
	// stop is only valid while powered on (409 otherwise, per spec).
	if !h.requirePowerState(w, r, logger, inst, instancedom.PowerStateOn) {
		return
	}
	inst.DesiredPowerState = instancedom.PowerStateOff
	h.applyPowerIntent(w, r, logger, inst, ifUnmodifiedSinceVersion(params.IfUnmodifiedSince))
}

// RestartInstance handles POST /v1/tenants/{tenant}/workspaces/{workspace}/instances/{name}/restart.
// Restarting is only valid while the instance is powered on; the delegator performs an on->off->on cycle.
func (h *Handler) RestartInstance(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdkcompute.RestartInstanceParams) {
	logger := h.Logger.With("provider", "compute", "resource", "instance", "name", name, "op", "restart")
	inst, ok := h.loadActiveInstance(w, r, logger, tenant, workspace, name)
	if !ok {
		return
	}
	// restart is only valid while powered on (409 otherwise, per spec).
	if !h.requirePowerState(w, r, logger, inst, instancedom.PowerStateOn) {
		return
	}
	// Start a durable restart: a fresh id plus the initial phase. Both annotations are written
	// in a single update; the delegator advances the phase and clears them when the cycle completes.
	restartID, err := newRestartID()
	if err != nil {
		frest.WriteErrorResponse(w, r, logger, err)
		return
	}
	inst.RestartID = restartID
	inst.RestartPhase = instancedom.RestartPhasePowerOff
	h.applyPowerIntent(w, r, logger, inst, ifUnmodifiedSinceVersion(params.IfUnmodifiedSince))
}

// currentPowerState returns the instance's power state, treating an unset value as off.
func currentPowerState(inst *instancedom.Instance) instancedom.PowerState {
	if inst.Status == nil || inst.Status.PowerState == "" {
		return instancedom.PowerStateOff
	}
	return inst.Status.PowerState
}

// requirePowerState writes a 409 Conflict and returns false when the instance is not in the
// power state a power action requires.
func (h *Handler) requirePowerState(w http.ResponseWriter, r *http.Request, logger *slog.Logger, inst *instancedom.Instance, want instancedom.PowerState) bool {
	if got := currentPowerState(inst); got != want {
		frest.WriteErrorResponse(w, r, logger, kernel.NewError(kernel.KindConflict,
			fmt.Errorf("instance %q power state is %q, action requires %q", inst.GetName(), got, want)))
		return false
	}
	return true
}

// loadActiveInstance loads the instance and verifies it is in the active state, which is a
// precondition for any power operation. It writes the appropriate error response and returns
// ok=false when the instance is missing or not active.
func (h *Handler) loadActiveInstance(w http.ResponseWriter, r *http.Request, logger *slog.Logger, tenant, workspace, name string) (*instancedom.Instance, bool) {
	id := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant, Workspace: workspace}}
	inst := newInstanceWithIdentity(id)
	if err := h.InstanceReader.Load(r.Context(), &inst); err != nil {
		frest.WriteErrorResponse(w, r, logger, err)
		return nil, false
	}
	if inst.Status == nil || inst.Status.State != commondomain.ResourceStateActive {
		frest.WriteErrorResponse(w, r, logger, kernel.NewError(kernel.KindConflict,
			fmt.Errorf("instance %q is not active", name)))
		return nil, false
	}
	return inst, true
}

// newRestartID returns a random opaque identifier for a restart request.
func newRestartID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("failed to generate restart id: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// ifUnmodifiedSinceVersion converts the optional If-Unmodified-Since precondition to a resource
// version string. An empty result means no precondition (fire-and-forget).
func ifUnmodifiedSinceVersion(v *sdkschema.IfUnmodifiedSince) string {
	if v == nil {
		return ""
	}
	return strconv.Itoa(*v)
}

// applyPowerIntent persists the power intent carried on inst and returns 202 Accepted. version is
// the optimistic-concurrency precondition from If-Unmodified-Since: when set, the update enforces
// it (a mismatch fails the request, consistent with create/update/delete); when empty, the update
// takes the conflict-retrying metadata path (fire-and-forget).
func (h *Handler) applyPowerIntent(w http.ResponseWriter, r *http.Request, logger *slog.Logger, inst *instancedom.Instance, version string) {
	inst.ResourceVersion = version
	if _, err := h.InstanceWriter.Update(r.Context(), inst); err != nil {
		frest.WriteErrorResponse(w, r, logger, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
