package rest

import (
	"log/slog"
	"net/http"
	"strconv"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/image/v1"
)

// ImageHandler implements the image methods of the storage group's
// sdkstorage.ServerInterface. It is embedded by the storage group owner
// (block-storage) Handler so the image methods are promoted onto it.
type ImageHandler struct {
	Reader persistencepkg.ReaderRepo[*imgdom.Image]
	Writer persistencepkg.WriterRepo[*imgdom.Image]
	Logger *slog.Logger
}

// ListImages handles GET /v1/tenants/{tenant}/images.
func (h *ImageHandler) ListImages(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkstorage.ListImagesParams) {
	logger := h.Logger.With("provider", "storage", "resource", "image")
	frest.HandleList(w, r, logger, ListParamsFromAPI(params, tenant), frest.ListerFromRepo(h.Reader), ImageIteratorToAPI)
}

// DeleteImage handles DELETE /v1/tenants/{tenant}/images/{name}.
func (h *ImageHandler) DeleteImage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkstorage.DeleteImageParams) {
	logger := h.Logger.With("provider", "storage", "resource", "image", "name", name)
	id := &ImageIdentity{name: name, tenant: tenant}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.Writer, newImageWithIdentity))
}

// GetImage handles GET /v1/tenants/{tenant}/images/{name}.
func (h *ImageHandler) GetImage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "storage", "resource", "image", "name", name)
	ir := &ImageIdentity{name: name, tenant: tenant}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.Reader, newImageWithIdentity), ImageToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateImage handles PUT /v1/tenants/{tenant}/images/{name}.
func (h *ImageHandler) CreateOrUpdateImage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkstorage.CreateOrUpdateImageParams) {
	logger := h.Logger.With("provider", "storage", "resource", "image", "name", name)
	id := &ImageIdentity{name: name, tenant: tenant}
	if params.IfUnmodifiedSince != nil {
		id.resourceVersion = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.Image, *imgdom.Image, *sdkschema.Image]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.Writer),
		Updater: frest.UpdaterFromRepo(h.Writer),
		APIToDomain: func(sdk sdkschema.Image, p persistencepkg.IdentifiableResource) *imgdom.Image {
			return ImageFromAPI(sdk, p.(*ImageIdentity), region)
		},
		DomainToAPI: ImageToAPIWithVerb(http.MethodPut),
	})
}
