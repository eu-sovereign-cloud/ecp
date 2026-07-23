package handler

import (
	"context"

	igwdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	igwk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	rtdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	rtk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	sgdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	sgrdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	sgrk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
	sgk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
)

// Aruba has no internet gateway and no route table resource: internet egress and routing are
// properties of the VPC itself, configured by Aruba when the VPC is created. Both SECA
// resources therefore have nothing to propagate and go active immediately, existing only so the
// SECA model stays consistent across providers.
//
// The InternetGateway doubles as a precondition of the SECA Network (see NetworkHandler), which
// is why it is modelled at all rather than rejected. Route specs are never propagated - see
// csp/aruba/README.md.
//
// SecurityGroup and SecurityGroupRule are no-ops for a different reason: they carry no VPC and
// nothing to attach to yet. A SECA SecurityGroup takes effect only when a NIC/compute instance
// references it (network.v1 nic/instance specs), and the standalone SecurityGroupRule is a
// reusable template that does nothing until a SecurityGroup pulls it in via ruleRefs. The Aruba
// SecurityGroup/SecurityRule (both of which require a VPCReference) are therefore created later,
// by the NIC/compute-instance work, in the specific VPC the attachment names - not here. See
// csp/aruba/README.md.

// The NIC is a no-op too: Aruba has no standalone NIC resource - a NIC's subnet and security
// groups are attributes of the CloudServer it belongs to. The SECA NIC therefore propagates
// nothing and goes active immediately; the compute-instance handler reads the NIC CRs to learn
// which subnet and security groups a CloudServer must be wired to. See csp/aruba/README.md.

var (
	_ igwk8s.InternetGatewayPlugin   = (*InternetGatewayHandler)(nil)
	_ rtk8s.RouteTablePlugin         = (*RouteTableHandler)(nil)
	_ sgk8s.SecurityGroupPlugin      = (*SecurityGroupHandler)(nil)
	_ sgrk8s.SecurityGroupRulePlugin = (*SecurityGroupRuleHandler)(nil)
	_ nick8s.NicPlugin               = (*NicHandler)(nil)
)

// InternetGatewayHandler accepts every InternetGateway without creating an Aruba resource.
type InternetGatewayHandler struct{}

func NewInternetGatewayHandler() *InternetGatewayHandler { return &InternetGatewayHandler{} }

func (h *InternetGatewayHandler) Create(_ context.Context, _ *igwdom.InternetGateway) error {
	return nil
}

func (h *InternetGatewayHandler) Delete(_ context.Context, _ *igwdom.InternetGateway) error {
	return nil
}

// RouteTableHandler accepts every RouteTable without creating an Aruba resource.
type RouteTableHandler struct{}

func NewRouteTableHandler() *RouteTableHandler { return &RouteTableHandler{} }

func (h *RouteTableHandler) Create(_ context.Context, _ *rtdom.RouteTable) error { return nil }

func (h *RouteTableHandler) Delete(_ context.Context, _ *rtdom.RouteTable) error { return nil }

// SecurityGroupHandler accepts every SecurityGroup without creating an Aruba resource. The Aruba
// SecurityGroup is created later, per VPC, by the NIC/compute-instance work that binds it.
type SecurityGroupHandler struct{}

func NewSecurityGroupHandler() *SecurityGroupHandler { return &SecurityGroupHandler{} }

func (h *SecurityGroupHandler) Create(_ context.Context, _ *sgdom.SecurityGroup) error { return nil }

func (h *SecurityGroupHandler) Delete(_ context.Context, _ *sgdom.SecurityGroup) error { return nil }

// SecurityGroupRuleHandler accepts every SecurityGroupRule without creating an Aruba resource. A
// standalone rule is a reusable template with no group or VPC, so there is nothing to propagate.
type SecurityGroupRuleHandler struct{}

func NewSecurityGroupRuleHandler() *SecurityGroupRuleHandler { return &SecurityGroupRuleHandler{} }

func (h *SecurityGroupRuleHandler) Create(_ context.Context, _ *sgrdom.SecurityGroupRule) error {
	return nil
}

func (h *SecurityGroupRuleHandler) Delete(_ context.Context, _ *sgrdom.SecurityGroupRule) error {
	return nil
}

// NicHandler accepts every NIC without creating an Aruba resource. Aruba has no standalone NIC;
// the compute-instance handler reads the NIC CRs to wire the CloudServer to its subnet and
// security groups.
type NicHandler struct{}

func NewNicHandler() *NicHandler { return &NicHandler{} }

func (h *NicHandler) Create(_ context.Context, _ *nicdom.Nic) error { return nil }

func (h *NicHandler) Delete(_ context.Context, _ *nicdom.Nic) error { return nil }
