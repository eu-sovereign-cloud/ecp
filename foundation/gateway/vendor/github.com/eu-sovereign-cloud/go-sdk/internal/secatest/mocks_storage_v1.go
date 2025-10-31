package secatest

import (
	"net/http"

	mockstorage "github.com/eu-sovereign-cloud/go-sdk/mock/spec/foundation.storage.v1"
	storage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/stretchr/testify/mock"
)

// Storage Sku
func MockListStorageSkusV1(sim *mockstorage.MockServerInterface, resp StorageSkuResponseV1) {
	sim.EXPECT().ListSkus(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, params storage.ListSkusParams) {
			if err := configGetHttpResponse(w, storageSkusResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockGetStorageSkusV1(sim *mockstorage.MockServerInterface, resp StorageSkuResponseV1) {
	sim.EXPECT().GetSku(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, name schema.ResourcePathParam) {
			if err := configGetHttpResponse(w, storageSkuResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

// Block Storage
func MockListBlockStoragesV1(sim *mockstorage.MockServerInterface, resp BlockStorageResponseV1) {
	sim.EXPECT().ListBlockStorages(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, params storage.ListBlockStoragesParams) {
			if err := configGetHttpResponse(w, blockStoragesResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockGetBlockStorageV1(sim *mockstorage.MockServerInterface, resp BlockStorageResponseV1) {
	sim.EXPECT().GetBlockStorage(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, name schema.ResourcePathParam) {
			if err := configGetHttpResponse(w, blockStorageResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockCreateOrUpdateBlockStorageV1(sim *mockstorage.MockServerInterface, resp BlockStorageResponseV1) {
	sim.EXPECT().CreateOrUpdateBlockStorage(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, name schema.ResourcePathParam, params storage.CreateOrUpdateBlockStorageParams) {
			if err := configPutHttpResponse(w, blockStorageResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockDeleteBlockStorageV1(sim *mockstorage.MockServerInterface) {
	sim.EXPECT().DeleteBlockStorage(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, workspace schema.WorkspacePathParam, name schema.ResourcePathParam, params storage.DeleteBlockStorageParams) {
			configDeleteHttpResponse(w)
		})
}

// Image
func MockListStorageImagesV1(sim *mockstorage.MockServerInterface, resp ImageResponseV1) {
	sim.EXPECT().ListImages(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, params storage.ListImagesParams) {
			if err := configGetHttpResponse(w, imagesResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockGetStorageImageV1(sim *mockstorage.MockServerInterface, resp ImageResponseV1) {
	sim.EXPECT().GetImage(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, name schema.ResourcePathParam) {
			if err := configGetHttpResponse(w, imageResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockCreateOrUpdateImageV1(sim *mockstorage.MockServerInterface, resp ImageResponseV1) {
	sim.EXPECT().CreateOrUpdateImage(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, name schema.ResourcePathParam, params storage.CreateOrUpdateImageParams) {
			if err := configPutHttpResponse(w, imageResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockDeleteImageV1(sim *mockstorage.MockServerInterface) {
	sim.EXPECT().DeleteImage(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, tenant schema.TenantPathParam, name schema.ResourcePathParam, params storage.DeleteImageParams) {
			configDeleteHttpResponse(w)
		})
}
