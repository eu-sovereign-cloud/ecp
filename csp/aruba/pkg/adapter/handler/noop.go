package handler

import (
	"context"

	igwdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	igwk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	rtdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	rtk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	sgrdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	sgrk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
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
// SecurityGroupRule is a no-op for a different reason: it carries no VPC and nothing to attach to
// yet. The standalone SecurityGroupRule is a reusable template that does nothing until a
// SecurityGroup pulls it in via ruleRefs; the Aruba SecurityRule (which requires a VPCReference)
// is created later, by the compute-instance work, in the specific VPC the attachment names - not
// here. The SecurityGroup itself is no longer a no-op: it accepts on create but reaps its
// materialised Aruba resources on delete, so it lives in security-group.go. See csp/aruba/README.md.

// The NIC is a no-op too: Aruba has no standalone NIC resource - a NIC's subnet and security
// groups are attributes of the CloudServer it belongs to. The SECA NIC therefore propagates
// nothing and goes active immediately; the compute-instance handler reads the NIC CRs to learn
// which subnet and security groups a CloudServer must be wired to. See csp/aruba/README.md.

// The Image is a no-op as well: Aruba has no image object. A SECA image is only a name; the boot
// volume that is created from it carries the matching Aruba OS template code (mapped in the
// block-storage handler via skumap), so the image resource has nothing to propagate and goes active
// immediately, existing only so the SECA boot-from-image model stays consistent. See
// csp/aruba/README.md.

var (
	_ igwk8s.InternetGatewayPlugin   = (*InternetGatewayHandler)(nil)
	_ rtk8s.RouteTablePlugin         = (*RouteTableHandler)(nil)
	_ sgrk8s.SecurityGroupRulePlugin = (*SecurityGroupRuleHandler)(nil)
	_ nick8s.NicPlugin               = (*NicHandler)(nil)
	_ imgk8s.ImagePlugin             = (*ImageHandler)(nil)
)

// ImageHandler accepts every Image without creating an Aruba resource. Aruba has no image object;
// the OS template code the image names is applied to the boot volume by the block-storage handler.
type ImageHandler struct{}

func NewImageHandler() *ImageHandler { return &ImageHandler{} }

func (h *ImageHandler) Create(_ context.Context, _ *imgdom.Image) error { return nil }

func (h *ImageHandler) Delete(_ context.Context, _ *imgdom.Image) error { return nil }

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

// Update is a no-op on every handler in this file, for the same reason Create is: none of these
// SECA resources has an Aruba object behind it, so there is nothing to carry a change to. Their
// labels reach Aruba only indirectly, through the resources that do exist - a NIC's labels never
// appear anywhere, while an image's OS template code is stamped on the boot volume that the
// block-storage handler retags in its own right.
func (h *ImageHandler) Update(_ context.Context, _ *imgdom.Image) error { return nil }

func (h *InternetGatewayHandler) Update(_ context.Context, _ *igwdom.InternetGateway) error {
	return nil
}

func (h *RouteTableHandler) Update(_ context.Context, _ *rtdom.RouteTable) error { return nil }

// SecurityGroupRuleHandler.Update is a no-op with a sharper edge than the others: a standalone
// rule's tags are stamped onto the Aruba SecurityRules built from it at instance-attach time, and
// those rules are immutable in place here - the handler cannot find them, since they are named and
// labelled after the materialised group rather than the rule. Editing a standalone rule's labels
// therefore does not reach rules already materialised from it. See csp/aruba/README.md.
func (h *SecurityGroupRuleHandler) Update(_ context.Context, _ *sgrdom.SecurityGroupRule) error {
	return nil
}

func (h *NicHandler) Update(_ context.Context, _ *nicdom.Nic) error { return nil }
