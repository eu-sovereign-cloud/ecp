// Package rest provides REST↔domain conversion and HTTP handlers for the storage API group.
package rest

import (
	"log/slog"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"

	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku"
)

// Handler is the HTTP handler for the storage API group.
// Block-storage methods are in block_storage_handler.go, image methods are in
// image_handler.go, and SKU methods are in storage_sku_handler.go.
type Handler struct {
	BlockStorageReader persistencepkg.ReaderRepo[*bsdom.BlockStorage]
	BlockStorageWriter persistencepkg.WriterRepo[*bsdom.BlockStorage]
	ImageReader        persistencepkg.ReaderRepo[*imgdom.Image]
	ImageWriter        persistencepkg.WriterRepo[*imgdom.Image]
	SKUReader          persistencepkg.ReaderRepo[*skudom.StorageSKU]
	Logger             *slog.Logger
}

var _ sdkstorage.ServerInterface = (*Handler)(nil)
