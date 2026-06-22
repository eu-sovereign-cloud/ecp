package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resources/common/frontend"
	skudom "github.com/eu-sovereign-cloud/ecp/resources/regional/network/network-skus/v1/domain"
	skurest "github.com/eu-sovereign-cloud/ecp/resources/regional/network/network-skus/v1/frontend/rest"
	netdom "github.com/eu-sovereign-cloud/ecp/resources/regional/network/networks/v1/domain"
)

// Handler is the HTTP handler for network resources (networks + SKUs).
// It implements the full sdknetwork.ServerInterface.
type Handler struct {
	NetworkReader persistence.ReaderRepo[*netdom.NetworkDomain]
	NetworkWriter persistence.WriterRepo[*netdom.NetworkDomain]
	SKUReader     persistence.ReaderRepo[*skudom.NetworkSKUDomain]
	Logger        *slog.Logger
}

var _ sdknetwork.ServerInterface = (*Handler)(nil)

// --- SKUs ---

func (h *Handler) ListSkus(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdknetwork.ListSkusParams) {
	logger := h.Logger.With("provider", "network", "resource", "sku")

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}
	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	listParams := resource.ListParams{
		Scope:     resource.Scope{Tenant: tenant},
		Limit:     validation.GetLimit(params.Limit),
		SkipToken: skipToken,
		Selector:  selector,
	}

	var domains []*skudom.NetworkSKUDomain
	nextSkipToken, err := h.SKUReader.List(r.Context(), listParams, &domains)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to list network SKUs", slog.Any("error", err))
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	writeJSON(w, r, logger, skurest.NetworkSKUDomainToAPIIterator(domains, nextSkipToken))
}

func (h *Handler) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "sku", "name", name)

	domain := &skudom.NetworkSKUDomain{}
	domain.Name = name
	domain.Tenant = tenant

	if err := h.SKUReader.Load(r.Context(), &domain); err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	writeJSON(w, r, logger, skurest.NetworkSKUDomainToAPI(domain))
}

// --- Networks ---

func (h *Handler) ListNetworks(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListNetworksParams) {
	logger := h.Logger.With("provider", "network", "resource", "network")
	listParams := ListParamsFromAPI(params, tenant, workspace)

	var domains []*netdom.NetworkDomain
	nextSkipToken, err := h.NetworkReader.List(r.Context(), listParams, &domains)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to list networks", slog.Any("error", err))
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	writeJSON(w, r, logger, NetworkDomainToAPIIterator(domains, nextSkipToken))
}

func (h *Handler) DeleteNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteNetworkParams) {
	logger := h.Logger.With("provider", "network", "resource", "network", "name", name)

	domain := &netdom.NetworkDomain{}
	domain.Name = name
	domain.Tenant = tenant
	domain.Workspace = workspace
	if params.IfUnmodifiedSince != nil {
		domain.ResourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}

	if err := h.NetworkWriter.Delete(r.Context(), domain); err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "network", "name", name)

	domain := &netdom.NetworkDomain{}
	domain.Name = name
	domain.Tenant = tenant
	domain.Workspace = workspace

	if err := h.NetworkReader.Load(r.Context(), &domain); err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	toAPI := NetworkDomainToAPIWithVerb(http.MethodGet)
	writeJSON(w, r, logger, toAPI(domain))
}

func (h *Handler) CreateOrUpdateNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateNetworkParams) {
	logger := h.Logger.With("provider", "network", "resource", "network", "name", name)

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

	var apiObj sdkschema.Network
	if err := json.Unmarshal(body, &apiObj); err != nil {
		commonfrontend.WriteErrorResponse(w, r, logger, err)
		return
	}

	id := &NetworkIdentity{
		name:            name,
		tenant:          tenant,
		workspace:       workspace,
		resourceVersion: resourceVersion,
	}
	region := frameworkconfig.Singleton().Region()
	domainObj := APIToNetworkDomain(apiObj, id, region)

	var result *netdom.NetworkDomain
	if resourceVersion == "" {
		r2, err := h.NetworkWriter.Create(r.Context(), domainObj)
		if err != nil {
			commonfrontend.WriteErrorResponse(w, r, logger, err)
			return
		}
		result = *r2
	} else {
		r2, err := h.NetworkWriter.Update(r.Context(), domainObj)
		if err != nil {
			commonfrontend.WriteErrorResponse(w, r, logger, err)
			return
		}
		result = *r2
	}

	toAPI := NetworkDomainToAPIWithVerb(http.MethodPut)
	writeJSON(w, r, logger, toAPI(result))
}

// --- Unimplemented (TODO) ---

func (h *Handler) ListInternetGateways(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListInternetGatewaysParams) {
	h.Logger.DebugContext(r.Context(), "ListInternetGateways not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) DeleteInternetGateway(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteInternetGatewayParams) {
	h.Logger.DebugContext(r.Context(), "DeleteInternetGateway not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) GetInternetGateway(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	h.Logger.DebugContext(r.Context(), "GetInternetGateway not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) CreateOrUpdateInternetGateway(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateInternetGatewayParams) {
	h.Logger.DebugContext(r.Context(), "CreateOrUpdateInternetGateway not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) ListRouteTables(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, params sdknetwork.ListRouteTablesParams) {
	h.Logger.DebugContext(r.Context(), "ListRouteTables not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) DeleteRouteTable(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteRouteTableParams) {
	h.Logger.DebugContext(r.Context(), "DeleteRouteTable not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) GetRouteTable(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam) {
	h.Logger.DebugContext(r.Context(), "GetRouteTable not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) CreateOrUpdateRouteTable(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateRouteTableParams) {
	h.Logger.DebugContext(r.Context(), "CreateOrUpdateRouteTable not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) ListSubnets(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, params sdknetwork.ListSubnetsParams) {
	h.Logger.DebugContext(r.Context(), "ListSubnets not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) DeleteSubnet(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteSubnetParams) {
	h.Logger.DebugContext(r.Context(), "DeleteSubnet not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) GetSubnet(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam) {
	h.Logger.DebugContext(r.Context(), "GetSubnet not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) CreateOrUpdateSubnet(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, network sdkschema.NetworkPathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateSubnetParams) {
	h.Logger.DebugContext(r.Context(), "CreateOrUpdateSubnet not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) ListNics(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListNicsParams) {
	h.Logger.DebugContext(r.Context(), "ListNics not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) DeleteNic(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteNicParams) {
	h.Logger.DebugContext(r.Context(), "DeleteNic not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) GetNic(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	h.Logger.DebugContext(r.Context(), "GetNic not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) CreateOrUpdateNic(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateNicParams) {
	h.Logger.DebugContext(r.Context(), "CreateOrUpdateNic not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) ListPublicIps(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListPublicIpsParams) {
	h.Logger.DebugContext(r.Context(), "ListPublicIps not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) DeletePublicIp(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeletePublicIpParams) {
	h.Logger.DebugContext(r.Context(), "DeletePublicIp not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) GetPublicIp(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	h.Logger.DebugContext(r.Context(), "GetPublicIp not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) CreateOrUpdatePublicIp(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdatePublicIpParams) {
	h.Logger.DebugContext(r.Context(), "CreateOrUpdatePublicIp not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) ListSecurityGroupRules(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListSecurityGroupRulesParams) {
	h.Logger.DebugContext(r.Context(), "ListSecurityGroupRules not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) DeleteSecurityGroupRule(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteSecurityGroupRuleParams) {
	h.Logger.DebugContext(r.Context(), "DeleteSecurityGroupRule not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) GetSecurityGroupRule(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	h.Logger.DebugContext(r.Context(), "GetSecurityGroupRule not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) CreateOrUpdateSecurityGroupRule(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateSecurityGroupRuleParams) {
	h.Logger.DebugContext(r.Context(), "CreateOrUpdateSecurityGroupRule not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) ListSecurityGroups(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListSecurityGroupsParams) {
	h.Logger.DebugContext(r.Context(), "ListSecurityGroups not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) DeleteSecurityGroup(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteSecurityGroupParams) {
	h.Logger.DebugContext(r.Context(), "DeleteSecurityGroup not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) GetSecurityGroup(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	h.Logger.DebugContext(r.Context(), "GetSecurityGroup not implemented")
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) CreateOrUpdateSecurityGroup(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateSecurityGroupParams) {
	h.Logger.DebugContext(r.Context(), "CreateOrUpdateSecurityGroup not implemented")
	w.WriteHeader(http.StatusNotImplemented)
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
