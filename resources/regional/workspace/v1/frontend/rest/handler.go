package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resources/common/frontend"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1"
)

// Handler is the HTTP handler for workspace resources.
// It implements the full sdkworkspace.ServerInterface.
type Handler struct {
	Reader persistence.ReaderRepo[*wsdom.WorkspaceDomain]
	Writer persistence.WriterRepo[*wsdom.WorkspaceDomain]
	Logger *slog.Logger
}

var _ sdkworkspace.ServerInterface = (*Handler)(nil)

// ListWorkspaces handles GET /v1/tenants/{tenant}/workspaces.
func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkworkspace.ListWorkspacesParams) {
	logger := h.Logger.With("provider", "workspace", "resource", "workspace")
	listParams := ListParamsFromAPI(params, tenant)

	var domains []*wsdom.WorkspaceDomain
	nextSkipToken, err := h.Reader.List(r.Context(), listParams, &domains)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to list workspaces", slog.Any("error", err))
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	writeJSON(w, r, logger, DomainToAPIIterator(domains, nextSkipToken))
}

// DeleteWorkspace handles DELETE /v1/tenants/{tenant}/workspaces/{name}.
func (h *Handler) DeleteWorkspace(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkworkspace.DeleteWorkspaceParams) {
	logger := h.Logger.With("provider", "workspace", "resource", "workspace", "name", name)

	domain := &wsdom.WorkspaceDomain{}
	domain.Name = name
	domain.Tenant = tenant
	if params.IfUnmodifiedSince != nil {
		domain.ResourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}

	if err := h.Writer.Delete(r.Context(), domain); err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// GetWorkspace handles GET /v1/tenants/{tenant}/workspaces/{name}.
func (h *Handler) GetWorkspace(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "workspace", "resource", "workspace", "name", name)

	domain := &wsdom.WorkspaceDomain{}
	domain.Name = name
	domain.Tenant = tenant

	if err := h.Reader.Load(r.Context(), &domain); err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	toAPI := DomainToAPIWithVerb(http.MethodGet)
	writeJSON(w, r, logger, toAPI(domain))
}

// CreateOrUpdateWorkspace handles PUT /v1/tenants/{tenant}/workspaces/{name}.
func (h *Handler) CreateOrUpdateWorkspace(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkworkspace.CreateOrUpdateWorkspaceParams) {
	logger := h.Logger.With("provider", "workspace", "resource", "workspace", "name", name)

	var resourceVersion string
	if params.IfUnmodifiedSince != nil {
		resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}
	defer func() { _ = r.Body.Close() }()

	var apiObj sdkschema.Workspace
	if err := json.Unmarshal(body, &apiObj); err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	id := &WorkspaceIdentity{
		name:            name,
		tenant:          tenant,
		resourceVersion: resourceVersion,
	}
	region := frameworkconfig.Singleton().Region()
	domainObj := APIToDomain(apiObj, id, region)

	var result *wsdom.WorkspaceDomain
	if resourceVersion == "" {
		r2, err := h.Writer.Create(r.Context(), domainObj)
		if err != nil {
			commonfrontend.WriteErrorResponse(w, r, logger, err)
			return
		}
		result = *r2
	} else {
		r2, err := h.Writer.Update(r.Context(), domainObj)
		if err != nil {
			commonfrontend.WriteErrorResponse(w, r, logger, err)
			return
		}
		result = *r2
	}

	toAPI := DomainToAPIWithVerb(http.MethodPut)
	writeJSON(w, r, logger, toAPI(result))
}

// writeJSON encodes v to JSON and writes it to w.
func writeJSON(w http.ResponseWriter, r *http.Request, logger *slog.Logger, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		logger.ErrorContext(r.Context(), "failed to encode response", slog.Any("error", err))
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}
	w.Header().Set("Content-Type", string(sdkschema.AcceptHeaderJson))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
