// Package rest provides REST↔domain conversion and HTTP handlers for the network API group.
package rest

import (
	"log/slog"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"

	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"
)

// resourceFormat formats a resource path string.
const resourceFormat = "%s/%s"

// workspaceScopedResourceFormat formats a workspace-scoped resource ref.
const workspaceScopedResourceFormat = "tenants/%s/workspaces/%s/providers/%s/%s"

// Handler is the HTTP handler for the network API group.
// Network and SKU methods are in network_handler.go / network_sku_handler.go.
// Stubs for unimplemented resources (internet-gateway, route-table, subnet, nic,
// public-ip, security-group, security-group-rule) are also in network_handler.go.
type Handler struct {
	NetworkReader persistencepkg.ReaderRepo[*netdom.Network]
	NetworkWriter persistencepkg.WriterRepo[*netdom.Network]
	SKUReader     persistencepkg.ReaderRepo[*skudom.NetworkSKU]
	Logger        *slog.Logger
}

var _ sdknetwork.ServerInterface = (*Handler)(nil)
