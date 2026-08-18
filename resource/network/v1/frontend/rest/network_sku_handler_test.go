package rest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"
)

// fakeSKURepo is a hand-rolled ReaderRepo for exercising the network Handler's SKU endpoints over
// HTTP. Load returns a copy of loadResult (or loadErr); List returns listResult (or listErr) and
// captures the params it was called with.
type fakeSKURepo struct {
	loadResult *skudom.NetworkSKU
	loadErr    error
	listResult []*skudom.NetworkSKU
	listErr    error
	listToken  *string
	gotParams  resource.ListFilter
}

func (f *fakeSKURepo) List(_ context.Context, params resource.ListFilter, list *[]*skudom.NetworkSKU) (*string, error) {
	f.gotParams = params
	if f.listErr != nil {
		return nil, f.listErr
	}
	*list = f.listResult
	return f.listToken, nil
}

func (f *fakeSKURepo) Load(_ context.Context, m **skudom.NetworkSKU) error {
	if f.loadErr != nil {
		return f.loadErr
	}
	cp := *f.loadResult
	*m = &cp
	return nil
}

func newSKUTestHandler(repo *fakeSKURepo) *Handler {
	return &Handler{
		SKUReader: repo,
		Logger:    slog.Default(),
	}
}

func TestHandler_GetSku(t *testing.T) {
	t.Run("found returns 200 with the mapped SKU", func(t *testing.T) {
		repo := &fakeSKURepo{loadResult: newNetworkSKU()}
		h := newSKUTestHandler(repo)

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		h.GetSku(rec, req, "t1", "sku1")

		require.Equal(t, http.StatusOK, rec.Code)
		var out sdkschema.NetworkSku
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
		require.Equal(t, "sku1", out.Metadata.Name)
		require.Equal(t, "t1", out.Metadata.Tenant)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		repo := &fakeSKURepo{loadErr: kernel.ErrNotFound}
		h := newSKUTestHandler(repo)

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		h.GetSku(rec, req, "t1", "missing")

		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHandler_ListSkus(t *testing.T) {
	t.Run("returns 200 with the mapped iterator", func(t *testing.T) {
		repo := &fakeSKURepo{listResult: []*skudom.NetworkSKU{newNetworkSKU()}}
		h := newSKUTestHandler(repo)

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		h.ListSkus(rec, req, "t1", sdknetwork.ListSkusParams{})

		require.Equal(t, http.StatusOK, rec.Code)
		var out sdknetwork.SkuIterator
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
		require.Len(t, out.Items, 1)
		require.Equal(t, "list", out.Metadata.Verb)
	})

	t.Run("forwards limit, skip token and label selector to the repo", func(t *testing.T) {
		repo := &fakeSKURepo{listResult: nil}
		h := newSKUTestHandler(repo)

		skipToken := "next"
		labels := "env=prod"
		limit := 50
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		h.ListSkus(rec, req, "t1", sdknetwork.ListSkusParams{
			SkipToken: &skipToken,
			Labels:    &labels,
			Limit:     &limit,
		})

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "t1", repo.gotParams.GetTenant())
		require.Equal(t, 50, repo.gotParams.GetLimit())
		require.Equal(t, skipToken, repo.gotParams.GetSkipToken())
		require.Equal(t, labels, repo.gotParams.GetSelector())
	})

	t.Run("repo error surfaces as 500", func(t *testing.T) {
		repo := &fakeSKURepo{listErr: kernel.ErrUnavailable}
		h := newSKUTestHandler(repo)

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		h.ListSkus(rec, req, "t1", sdknetwork.ListSkusParams{})

		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
