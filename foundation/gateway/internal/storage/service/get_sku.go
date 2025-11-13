package service

import (
	"net/http"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/storage/controller"
)

type GetSku struct {
	get_sku controller.GetSKU
}

func (s *GetSku) GetSku(w http.ResponseWriter, r *http.Request, tenant externalRef0.TenantPathParam, name externalRef0.ResourcePathParam) {
	// handle the http to controller and back again
}
