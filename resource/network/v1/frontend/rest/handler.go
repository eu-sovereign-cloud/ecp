// Package rest provides REST↔domain conversion and HTTP handlers for the network API group.
package rest

import (
	"log/slog"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"

	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

// resourceFormat formats a resource path string.
const resourceFormat = "%s/%s"

// workspaceScopedResourceFormat formats a workspace-scoped resource ref.
const workspaceScopedResourceFormat = "tenants/%s/workspaces/%s/providers/%s/%s"

// Handler is the HTTP handler for the network API group.
// Network and SKU methods are in network_handler.go / network_sku_handler.go;
// NIC methods are in nic_handler.go.
// Stubs for unimplemented resources (internet-gateway, route-table, subnet,
// public-ip, security-group, security-group-rule) are in network_handler.go.
type Handler struct {
	NetworkReader persistencepkg.ReaderRepo[*netdom.Network]
	NetworkWriter persistencepkg.WriterRepo[*netdom.Network]
	SKUReader     persistencepkg.ReaderRepo[*skudom.NetworkSKU]
	NicReader     persistencepkg.ReaderRepo[*nicdom.Nic]
	NicWriter     persistencepkg.WriterRepo[*nicdom.Nic]
	Logger        *slog.Logger
}

var _ sdknetwork.ServerInterface = (*Handler)(nil)
