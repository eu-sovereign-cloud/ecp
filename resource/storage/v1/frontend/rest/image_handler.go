package rest

import (
	"net/http"
	"strconv"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	frameworkconfig "github.com/eu-sovereign-cloud/ecp/framework/frontend/config"
	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
)

// ListImages handles GET /v1/tenants/{tenant}/images.
func (h *Handler) ListImages(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, params sdkstorage.ListImagesParams) {
	logger := h.Logger.With("provider", "storage", "resource", "image")
	frest.HandleList(w, r, logger, imageListParamsFromAPI(params, tenant), frest.ListerFromRepo(h.ImageReader), imageIteratorToAPI)
}

// DeleteImage handles DELETE /v1/tenants/{tenant}/images/{name}.
func (h *Handler) DeleteImage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkstorage.DeleteImageParams) {
	logger := h.Logger.With("provider", "storage", "resource", "image")
	id := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant}}
	if params.IfUnmodifiedSince != nil {
		id.Version = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	frest.HandleDelete(w, r, logger, id, frest.DeleterFromRepo(h.ImageWriter, newImageWithIdentity))
}

// GetImage handles GET /v1/tenants/{tenant}/images/{name}.
func (h *Handler) GetImage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam) {
	logger := h.Logger.With("provider", "storage", "resource", "image")
	ir := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant}}
	frest.HandleGet(w, r, logger, ir, frest.GetterFromRepo(h.ImageReader, newImageWithIdentity), imageToAPIWithVerb(http.MethodGet))
}

// CreateOrUpdateImage handles PUT /v1/tenants/{tenant}/images/{name}.
func (h *Handler) CreateOrUpdateImage(w http.ResponseWriter, r *http.Request, tenant sdkschema.TenantPathParam, name sdkschema.ResourcePathParam, params sdkstorage.CreateOrUpdateImageParams) {
	logger := h.Logger.With("provider", "storage", "resource", "image")
	id := &resource.Identity{Name: name, Scope: resource.Scope{Tenant: tenant}}
	if params.IfUnmodifiedSince != nil {
		id.Version = strconv.Itoa(*params.IfUnmodifiedSince)
	}
	region := frameworkconfig.Singleton().Region()
	frest.HandleUpsert(w, r, logger, frest.UpsertOptions[sdkschema.Image, *imgdom.Image, *sdkschema.Image]{
		Params:  id,
		Creator: frest.CreatorFromRepo(h.ImageWriter),
		Updater: frest.UpdaterFromRepo(h.ImageWriter),
		APIToDomain: func(sdk sdkschema.Image, p persistencepkg.IdentifiableResource) (*imgdom.Image, error) {
			return imageFromAPI(sdk, p.(*resource.Identity), region)
		},
		DomainToAPI: imageToAPIWithVerb(http.MethodPut),
	})
}
