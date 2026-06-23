package rest

import (
	"log/slog"
	"net/http"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/network/network-sku/v1"
	skurest "github.com/eu-sovereign-cloud/ecp/resource/network/network-sku/v1/frontend/rest"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/network/v1"
)

// Handler is the HTTP handler for network resources (networks + SKUs).
// It implements the full sdknetwork.ServerInterface.
type Handler struct {
	NetworkReader persistencepkg.ReaderRepo[*netdom.Network]
	NetworkWriter persistencepkg.WriterRepo[*netdom.Network]
	SKUReader     persistencepkg.ReaderRepo[*skudom.NetworkSKU]
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
	frest.HandleList(w, r, logger, listParams, frest.ListerFromRepo(h.SKUReader), skurest.NetworkSKUDomainToAPIIterator)
}

func (h *Handler) GetSku(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "sku", "name", name)
	ir := &networkSKUIdentity{name: name, tenant: tenant}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.SKUReader, newNetworkSKUWithIdentity), skurest.NetworkSKUDomainToAPI)
}

// --- Networks ---

func (h *Handler) ListNetworks(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, params sdknetwork.ListNetworksParams) {
	logger := h.Logger.With("provider", "network", "resource", "network")
	frest.HandleList(w, r, logger, ListParamsFromAPI(params, tenant, workspace), frest.ListerFromRepo(h.NetworkReader), NetworkDomainToAPIIterator)
}

func (h *Handler) DeleteNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.DeleteNetworkParams) {
	logger := h.Logger.With("provider", "network", "resource", "network", "name", name)
	id := &NetworkIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.NetworkWriter, newNetworkWithIdentity))
}

func (h *Handler) GetNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "network", "resource", "network", "name", name)
	ir := &NetworkIdentity{name: name, tenant: tenant, workspace: workspace}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.NetworkReader, newNetworkWithIdentity), NetworkDomainToAPIWithVerb(http.MethodGet))
}

func (h *Handler) CreateOrUpdateNetwork(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, workspace sdkschema.WorkspacePathParam, name sdkschema.ResourcePathParam, params sdknetwork.CreateOrUpdateNetworkParams) {
	logger := h.Logger.With("provider", "network", "resource", "network", "name", name)
	id := &NetworkIdentity{name: name, tenant: tenant, workspace: workspace}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.Network, *netdom.Network, *sdkschema.Network]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.NetworkWriter),
		Updater: frest.UpdaterFromRepo(h.NetworkWriter),
		APIToDomain: func(sdk sdkschema.Network, p persistencepkg.IdentifiableResource) *netdom.Network {
			return APIToNetworkDomain(sdk, p.(*NetworkIdentity), region)
		},
		DomainToAPI: NetworkDomainToAPIWithVerb(http.MethodPut),
	})
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

// newNetworkWithIdentity returns a *netdom.Network populated with identity fields from ir.
func newNetworkWithIdentity(ir persistencepkg.IdentifiableResource) *netdom.Network {
	d := &netdom.Network{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.Workspace = ir.GetWorkspace()
	d.ResourceVersion = ir.GetVersion()
	return d
}

// networkSKUIdentity is a minimal IdentifiableResource for network-SKU get operations.
type networkSKUIdentity struct {
	name   string
	tenant string
}

func (s *networkSKUIdentity) GetName() string      { return s.name }
func (s *networkSKUIdentity) GetVersion() string   { return "" }
func (s *networkSKUIdentity) GetTenant() string    { return s.tenant }
func (s *networkSKUIdentity) GetWorkspace() string { return "" }

// newNetworkSKUWithIdentity returns a *skudom.NetworkSKU populated with identity fields from ir.
func newNetworkSKUWithIdentity(ir persistencepkg.IdentifiableResource) *skudom.NetworkSKU {
	d := &skudom.NetworkSKU{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	return d
}
