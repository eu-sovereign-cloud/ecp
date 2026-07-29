// Package rest provides REST↔domain conversion and HTTP handlers for the network API group.
package rest

import (
	"log/slog"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"

	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

// Handler is the HTTP handler for the network API group.
// Network and SKU methods are in network_handler.go / network_sku_handler.go;
// NIC methods are in nic_handler.go.
// PublicIp methods are in public_ip_handler.go.
// InternetGateway methods are in internet_gateway_handler.go.
// RouteTable methods are in route_table_handler.go.
// Subnet methods are in subnet_handler.go.
// SecurityGroup methods are in security_group_handler.go.
// SecurityGroupRule methods are in security_group_rule_handler.go.
type Handler struct {
	NetworkReader           persistencepkg.ReaderRepo[*netdom.Network]
	NetworkWriter           persistencepkg.WriterRepo[*netdom.Network]
	SKUReader               persistencepkg.ReaderRepo[*skudom.NetworkSKU]
	NicReader               persistencepkg.ReaderRepo[*nicdom.Nic]
	NicWriter               persistencepkg.WriterRepo[*nicdom.Nic]
	InternetGatewayReader   persistencepkg.ReaderRepo[*internetgatewaydom.InternetGateway]
	InternetGatewayWriter   persistencepkg.WriterRepo[*internetgatewaydom.InternetGateway]
	PublicIpReader          persistencepkg.ReaderRepo[*publicipdom.PublicIp]
	PublicIpWriter          persistencepkg.WriterRepo[*publicipdom.PublicIp]
	RouteTableReader        persistencepkg.ReaderRepo[*routetabledom.RouteTable]
	RouteTableWriter        persistencepkg.WriterRepo[*routetabledom.RouteTable]
	SubnetReader            persistencepkg.ReaderRepo[*subnetdom.Subnet]
	SubnetWriter            persistencepkg.WriterRepo[*subnetdom.Subnet]
	SecurityGroupReader     persistencepkg.ReaderRepo[*securitygroupdom.SecurityGroup]
	SecurityGroupWriter     persistencepkg.WriterRepo[*securitygroupdom.SecurityGroup]
	SecurityGroupRuleReader persistencepkg.ReaderRepo[*securitygroupruledom.SecurityGroupRule]
	SecurityGroupRuleWriter persistencepkg.WriterRepo[*securitygroupruledom.SecurityGroupRule]
	Logger                  *slog.Logger
}

var _ sdknetwork.ServerInterface = (*Handler)(nil)
