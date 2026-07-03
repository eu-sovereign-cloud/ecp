// Package rest provides REST↔domain conversion and HTTP handlers for the network API group.
package rest

import (
	"log/slog"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"

	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

// Handler is the HTTP handler for the network API group.
// Network and SKU methods are in network_handler.go / network_sku_handler.go;
// NIC methods are in nic_handler.go.
// PublicIp methods are in public_ip_handler.go.
// Stubs for unimplemented resources (internet-gateway, route-table, subnet,
// security-group, security-group-rule) are in network_handler.go.
type Handler struct {
	NetworkReader  persistencepkg.ReaderRepo[*netdom.Network]
	NetworkWriter  persistencepkg.WriterRepo[*netdom.Network]
	SKUReader      persistencepkg.ReaderRepo[*skudom.NetworkSKU]
	NicReader      persistencepkg.ReaderRepo[*nicdom.Nic]
	NicWriter      persistencepkg.WriterRepo[*nicdom.Nic]
	PublicIpReader persistencepkg.ReaderRepo[*publicipdom.PublicIp]
	PublicIpWriter persistencepkg.WriterRepo[*publicipdom.PublicIp]
	Logger         *slog.Logger
}

var _ sdknetwork.ServerInterface = (*Handler)(nil)
