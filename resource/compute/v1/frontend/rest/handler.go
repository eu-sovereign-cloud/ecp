// Package rest provides REST↔domain conversion and HTTP handlers for the compute API group.
package rest

import (
	"log/slog"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"

	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku"
)

// Handler is the HTTP handler for the compute API group.
// Instance CRUD and power-state (start/stop/restart) methods are in instance_handler.go;
// SKU read methods are in sku_handler.go.
type Handler struct {
	InstanceReader persistencepkg.ReaderRepo[*instancedom.Instance]
	InstanceWriter persistencepkg.WriterRepo[*instancedom.Instance]
	SKUReader      persistencepkg.ReaderRepo[*skudom.InstanceSKU]
	Logger         *slog.Logger
}

var _ sdkcompute.ServerInterface = (*Handler)(nil)
