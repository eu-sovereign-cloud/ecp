package secatest

import (
	"net/http"

	mockregion "github.com/eu-sovereign-cloud/go-sdk/mock/spec/foundation.region.v1"
	region "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"

	"github.com/stretchr/testify/mock"
)

// Region
func MockListRegionsV1(sim *mockregion.MockServerInterface, resp RegionResponseV1) {
	sim.EXPECT().ListRegions(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, lrp region.ListRegionsParams) {
			configHttpResponse(w, http.StatusOK)

			for i := range resp.Providers {
				resp.Providers[i].URL = "http://" + r.Host + resp.Providers[i].URL
			}

			if err := processTemplate(w, regionsTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func MockGetRegionV1(sim *mockregion.MockServerInterface, resp RegionResponseV1) {
	sim.EXPECT().GetRegion(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(w http.ResponseWriter, r *http.Request, name string) {
			configHttpResponse(w, http.StatusOK)

			for i := range resp.Providers {
				resp.Providers[i].URL = "http://" + r.Host + resp.Providers[i].URL
			}

			if err := processTemplate(w, regionResponseTemplateV1, resp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}
